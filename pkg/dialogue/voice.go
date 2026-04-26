package dialogue

import (
	"log"

	"github.com/hybridgroup/go-sayanything/pkg/say"
	"github.com/hybridgroup/go-sayanything/pkg/tts"
)

var names = []string{"llama3000", "gemmai", "phineas"}

type Voice struct {
	Name string
	t    tts.Speaker
	p    say.Player
}

func NewVoice(name, lang, voice, dataDir string, gpu bool) (*Voice, error) {
	t := tts.NewPiper(lang, voice)
	if err := t.Connect(dataDir); err != nil {
		return nil, err
	}

	if gpu {
		t.UseGPU(true)
	}

	p := say.NewPlayer("wav")

	return &Voice{Name: name, t: t, p: *p}, nil
}

var speaking = 0

// SayOnce speaks the given text synchronously, blocking until playback is complete.
func (v *Voice) SayOnce(what string) error {
	if len(what) == 0 {
		return nil
	}

	log.Printf("%s says: %s", v.Name, what)

	b, err := v.t.Speech(what)
	if err != nil {
		return err
	}

	return v.p.Say(b)
}

func (v *Voice) SayAnything(what string) error {
	if len(what) == 0 {
		return nil
	}

	log.Printf("%s says: %s", v.Name, what)

	data, err := v.t.Speech(what)
	if err != nil {
		return err
	}

	speaking++

	go func() {
		v.p.Say(data)
		speaking--
	}()

	return nil
}
