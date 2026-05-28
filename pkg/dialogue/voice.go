package dialogue

import (
	"log"
	"sync"

	"github.com/talkingheads2053/sayanything/pkg/say"
	"github.com/talkingheads2053/sayanything/pkg/tts"
)

var (
	sharedPlayer     *say.Player
	sharedPlayerOnce sync.Once
)

func getSharedPlayer() *say.Player {
	sharedPlayerOnce.Do(func() {
		sharedPlayer = say.NewPlayer("wav")
	})
	return sharedPlayer
}

type Voice struct {
	Name string
	t    tts.Speaker
	p    *say.Player
}

func NewVoice(name, lang, voice, dataDir string, gpu bool) (*Voice, error) {
	t := tts.NewPiper(lang, voice)
	if err := t.Connect(dataDir); err != nil {
		return nil, err
	}

	if gpu {
		t.UseGPU(true)
	}

	return &Voice{Name: name, t: t, p: getSharedPlayer()}, nil
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
