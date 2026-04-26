package hotmic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	portaudio "github.com/gordonklaus/portaudio"
	"golang.org/x/term"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

// rawWriter wraps an io.Writer and replaces bare \n with \r\n so that log
// output is rendered correctly while the terminal is in raw mode.
type rawWriter struct{ w io.Writer }

func (rw rawWriter) Write(p []byte) (int, error) {
	_, err := rw.w.Write(bytes.ReplaceAll(p, []byte("\n"), []byte("\r\n")))
	return len(p), err
}

const (
	numChannels  = 1
	framesPerBuf = 1024
)

// HotMic captures audio from the default microphone when a toggle key is
// pressed, then transcribes it using a local whisper.cpp model.
type HotMic struct {
	key      rune
	language string
	model    whisper.Model
}

// Options configures a HotMic instance.
type Options struct {
	// Key is the rune that toggles recording on/off. Defaults to space (' ').
	Key rune

	// ModelPath is the path to a whisper.cpp ggml model file (e.g. ggml-base.en.bin).
	ModelPath string

	// Language is the BCP-47 language code to use for transcription, or "auto"
	// for automatic language detection. Defaults to "auto".
	Language string
}

// New loads the whisper model at opts.ModelPath and returns a HotMic ready for
// use. Call Close when done.
func New(opts Options) (*HotMic, error) {
	if opts.Key == 0 {
		opts.Key = ' '
	}
	if opts.Language == "" {
		opts.Language = "auto"
	}

	model, err := whisper.New(opts.ModelPath)
	if err != nil {
		return nil, err
	}

	return &HotMic{
		key:      opts.Key,
		language: opts.Language,
		model:    model,
	}, nil
}

// Close releases the whisper model resources.
func (h *HotMic) Close() error {
	return h.model.Close()
}

// Listen opens /dev/tty for key reading, puts it into raw mode, and starts
// monitoring for the toggle key. Each time the key is pressed to stop a
// recording, the transcribed text is sent on the returned channel. The channel
// is closed when ctx is cancelled or an unrecoverable error occurs.
//
// Listen returns only after the terminal has been successfully set to raw mode,
// so any initialisation error is returned synchronously to the caller.
func (h *HotMic) Listen(ctx context.Context) (<-chan string, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("hotmic: portaudio init: %w", err)
	}

	// Open /dev/tty directly so that stdin redirections don't affect us.
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		_ = portaudio.Terminate()
		return nil, fmt.Errorf("hotmic: open /dev/tty: %w", err)
	}

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		tty.Close()
		_ = portaudio.Terminate()
		return nil, fmt.Errorf("hotmic: set raw terminal: %w", err)
	}

	out := make(chan string, 4)

	go func() {
		defer close(out)
		defer tty.Close()
		defer func() {
			if err := term.Restore(int(tty.Fd()), oldState); err != nil {
				log.Printf("hotmic: restore terminal: %v", err)
			}
		}()
		defer func() {
			if err := portaudio.Terminate(); err != nil {
				log.Printf("hotmic: portaudio terminate: %v", err)
			}
		}()

		// Redirect log output through rawWriter so \n becomes \r\n in raw mode.
		prevLogOut := log.Writer()
		log.SetOutput(rawWriter{prevLogOut})
		defer log.SetOutput(prevLogOut)

		keyCh := make(chan rune, 4)
		go func() {
			b := make([]byte, 1)
			for {
				n, err := tty.Read(b)
				if err != nil || n == 0 {
					return
				}
				select {
				case keyCh <- rune(b[0]):
				case <-ctx.Done():
					return
				}
			}
		}()

		var (
			recording   bool
			stopRecCh   chan struct{}
			collectedCh chan []float32
		)

		for {
			select {
			case <-ctx.Done():
				if recording {
					close(stopRecCh)
					<-collectedCh
				}
				return

			case k := <-keyCh:
				switch {
				case k == 3 || k == 4: // Ctrl+C or Ctrl+D
					if recording {
						close(stopRecCh)
						<-collectedCh
					}
					return

				case k == h.key:
					if !recording {
						stopRecCh = make(chan struct{})
						collectedCh = make(chan []float32, 1)
						go captureAudio(stopRecCh, collectedCh)
						recording = true
						log.Println("hotmic: recording...")
					} else {
						close(stopRecCh)
						samples := <-collectedCh
						recording = false
						log.Println("hotmic: transcribing...")

						if len(samples) == 0 {
							continue
						}
						text, err := h.transcribe(samples)
						if err != nil {
							log.Printf("hotmic: transcribe: %v", err)
							continue
						}
						if text == "" {
							continue
						}
						select {
						case out <- text:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return out, nil
}

// captureAudio records mono float32 audio from the default input device at
// whisper's required sample rate until stop is closed, then sends all
// collected samples on out.
func captureAudio(stop <-chan struct{}, out chan<- []float32) {
	buf := make([]float32, framesPerBuf)
	stream, err := portaudio.OpenDefaultStream(numChannels, 0, float64(whisper.SampleRate), framesPerBuf, buf)
	if err != nil {
		log.Printf("hotmic: open stream: %v", err)
		out <- nil
		return
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		log.Printf("hotmic: start stream: %v", err)
		out <- nil
		return
	}
	defer stream.Stop()

	var collected []float32
	for {
		select {
		case <-stop:
			out <- collected
			return
		default:
			if err := stream.Read(); err != nil {
				log.Printf("hotmic: stream read: %v", err)
				out <- collected
				return
			}
			tmp := make([]float32, len(buf))
			copy(tmp, buf)
			collected = append(collected, tmp...)
		}
	}
}

// transcribe converts raw mono float32 PCM samples (at whisper.SampleRate) to text.
func (h *HotMic) transcribe(samples []float32) (string, error) {
	wctx, err := h.model.NewContext()
	if err != nil {
		return "", err
	}
	if wctx.IsMultilingual() {
		if err := wctx.SetLanguage(h.language); err != nil {
			return "", err
		}
	}
	if err := wctx.Process(samples, nil, nil); err != nil {
		return "", err
	}

	var sb strings.Builder
	for {
		seg, err := wctx.NextSegment()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		sb.WriteString(seg.Text)
	}

	return strings.TrimSpace(sb.String()), nil
}
