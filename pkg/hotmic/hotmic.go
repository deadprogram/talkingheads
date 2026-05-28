package hotmic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/ardanlabs/bucky/pkg/whisper"
	portaudio "github.com/gordonklaus/portaudio"
	"golang.org/x/term"
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
	key        rune
	language   string
	deviceName string
	ctx        whisper.Context

	// recording state (protected by mu)
	mu        sync.Mutex
	stopRec   chan struct{}
	samplesCh chan []float32
}

// Options configures a HotMic instance.
type Options struct {
	// Key is the rune that toggles recording on/off. Defaults to space (' ').
	Key rune

	// LibPath is the path to the directory containing the libwhisper shared
	// library. If empty, the BUCKY_LIB environment variable is used.
	LibPath string

	// ModelPath is the path to a whisper.cpp ggml model file (e.g. ggml-base.en.bin).
	ModelPath string

	// Language is the BCP-47 language code to use for transcription, or "auto"
	// for automatic language detection. Defaults to "auto".
	Language string

	// DeviceName is a case-insensitive substring of the PortAudio input device
	// name to use for capture (e.g. "USB", "Yeti"). When empty the system
	// default input device is used.
	DeviceName string
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
	if opts.LibPath == "" {
		opts.LibPath = os.Getenv("BUCKY_LIB")
	}

	if err := whisper.Load(opts.LibPath); err != nil {
		return nil, err
	}

	cparams := whisper.ContextDefaultParams()
	ctx, err := whisper.InitFromFileWithParams(opts.ModelPath, cparams)
	if err != nil {
		return nil, err
	}

	return &HotMic{
		key:        opts.Key,
		language:   opts.Language,
		deviceName: opts.DeviceName,
		ctx:        ctx,
	}, nil
}

// Close releases the whisper model resources. Safe to call more than once.
func (h *HotMic) Close() error {
	whisper.Free(h.ctx)
	h.ctx = 0
	return nil
}

// StartCapture begins audio capture from the default microphone.
//
// The entire PortAudio lifecycle (Initialize → stream read loop → Terminate)
// runs on a single OS thread that is pinned with runtime.LockOSThread, which
// satisfies the requirement of audio backends (ALSA, PulseAudio, CoreAudio)
// that all calls happen on the same thread.
//
// StartCapture blocks only until PortAudio has been initialised; it returns
// an error if capture is already in progress or if PortAudio cannot start.
func (h *HotMic) StartCapture() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopRec != nil {
		return fmt.Errorf("hotmic: capture already in progress")
	}

	ready := make(chan error, 1)
	h.stopRec = make(chan struct{})
	h.samplesCh = make(chan []float32, 1)

	stop := h.stopRec
	samples := h.samplesCh
	deviceName := h.deviceName

	go func() {
		// Pin this goroutine to its OS thread so that portaudio.Initialize,
		// all stream operations, and portaudio.Terminate all run on the same
		// underlying thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := portaudio.Initialize(); err != nil {
			ready <- fmt.Errorf("hotmic: portaudio init: %w", err)
			return
		}
		ready <- nil

		captureAudio(stop, samples, deviceName)

		if err := portaudio.Terminate(); err != nil {
			log.Printf("hotmic: portaudio terminate: %v", err)
		}
	}()

	return <-ready
}

// StopCapture signals the capture goroutine to stop and blocks until the
// collected PCM samples are returned. portaudio.Terminate is called by the
// capture goroutine itself (on the same OS thread) after samples are
// delivered. Returns an error if no capture is in progress.
func (h *HotMic) StopCapture() ([]float32, error) {
	h.mu.Lock()
	stop := h.stopRec
	ch := h.samplesCh
	h.stopRec = nil
	h.samplesCh = nil
	h.mu.Unlock()

	if stop == nil {
		return nil, fmt.Errorf("hotmic: no capture in progress")
	}

	close(stop)
	samples := <-ch // wait for captureAudio to finish and portaudio to terminate
	return samples, nil
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
						go captureAudio(stopRecCh, collectedCh, h.deviceName)
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

// findInputDevice returns the first PortAudio input device whose name
// contains deviceName (case-insensitive). Must be called while PortAudio is
// initialised.
func findInputDevice(deviceName string) (*portaudio.DeviceInfo, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("enumerate devices: %w", err)
	}
	nameLower := strings.ToLower(deviceName)
	for _, d := range devices {
		if d.MaxInputChannels > 0 && strings.Contains(strings.ToLower(d.Name), nameLower) {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no input device found matching %q", deviceName)
}

// captureAudio records mono float32 audio from the given input device (or the
// system default when deviceName is empty) at whisper's required sample rate
// until stop is closed, then sends all collected samples on out.
func captureAudio(stop <-chan struct{}, out chan<- []float32, deviceName string) {
	buf := make([]float32, framesPerBuf)

	var stream *portaudio.Stream
	var err error
	if deviceName != "" {
		dev, devErr := findInputDevice(deviceName)
		if devErr != nil {
			log.Printf("hotmic: %v", devErr)
			out <- nil
			return
		}
		p := portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Device:   dev,
				Channels: numChannels,
				Latency:  dev.DefaultLowInputLatency,
			},
			SampleRate:      float64(whisper.SampleRate),
			FramesPerBuffer: framesPerBuf,
		}
		stream, err = portaudio.OpenStream(p, buf)
	} else {
		stream, err = portaudio.OpenDefaultStream(numChannels, 0, float64(whisper.SampleRate), framesPerBuf, buf)
	}
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

// Transcribe converts raw mono float32 PCM samples (at whisper.SampleRate) to text.
// It is safe to call concurrently with StartCapture/StopCapture.
func (h *HotMic) Transcribe(samples []float32) (string, error) {
	return h.transcribe(samples)
}

// transcribe is the internal implementation of Transcribe.
func (h *HotMic) transcribe(samples []float32) (string, error) {
	wparams := whisper.FullDefaultParams(whisper.SamplingGreedy)
	wparams.PrintProgress = 0
	wparams.PrintRealtime = 0
	wparams.PrintTimestamps = 0
	wparams.NoTimestamps = 1

	var refs whisper.StringRefs
	if whisper.IsMultilingual(h.ctx) {
		lang := h.language
		if lang == "auto" {
			lang = ""
		}
		if err := refs.SetLanguage(&wparams, lang); err != nil {
			return "", err
		}
	}
	defer refs.KeepAlive()

	if err := whisper.Full(h.ctx, wparams, samples); err != nil {
		return "", err
	}

	var sb strings.Builder
	for i := int32(0); i < whisper.FullNSegments(h.ctx); i++ {
		sb.WriteString(whisper.FullGetSegmentText(h.ctx, i))
	}

	return strings.TrimSpace(sb.String()), nil
}
