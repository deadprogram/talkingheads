package main

import (
	"strings"

	"github.com/hybridgroup/go-sayanything/pkg/say"
	"github.com/hybridgroup/go-sayanything/pkg/tts"
)

func StartSayingAnything(t tts.Speaker, p *say.Player, responses chan string) error {
	for text := range responses {
		err := SayAnything(t, p, text)
		if err != nil {
			return err
		}
	}

	return nil
}

var speaking = 0

func SayAnything(t tts.Speaker, p *say.Player, text string) error {
	if len(text) == 0 {
		return nil
	}

	println(text)

	data, err := t.Speech(text)
	if err != nil {
		return err
	}

	speaking++
	if sp != nil {
		sp.Write([]byte(led + "\r"))
	}

	go func() {
		p.Say(data)
		speaking--

		if sp != nil {
			if speaking == 0 {
				sp.Write([]byte("stop\r"))
			}
		}
	}()

	return nil
}

func SayAnythingOnce(t tts.Speaker, p *say.Player, text string) error {
	if len(text) == 0 {
		return nil
	}

	println(text)

	data, err := t.Speech(text)
	if err != nil {
		return err
	}

	speaking++
	if sp != nil {
		sp.Write([]byte(led + "\r"))
	}

	p.Say(data)
	speaking--

	if sp != nil {
		if speaking == 0 {
			sp.Write([]byte("stop\r"))
		}
	}

	return nil
}

func cleanupText(text, cleanup string) string {
	if strings.Contains(text, cleanup) {
		return strings.ReplaceAll(text, cleanup, "")
	}

	return text
}
