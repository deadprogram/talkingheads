package main

import (
	"strings"

	"github.com/hybridgroup/go-sayanything/pkg/say"
	"github.com/hybridgroup/go-sayanything/pkg/tts"
)

func StartSayingAnything(t tts.Speaker, p *say.Player, responses chan string, listening chan string) error {
	for {
		select {
		case text := <-responses:
			if err := SayAnything(t, p, text); err != nil {
				return err
			}
		case who := <-listening:
			Listen(who)
		}
	}
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

	// just in case the name is left in the text
	text = strings.TrimPrefix(strings.TrimSpace(text), name+":")
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

func Listen(who string) {
	if sp == nil {
		return
	}

	switch name {
	case "llama3000":
		// servo is reversed?
		sp.Write([]byte("left\r"))
	case "phineas":
		switch who {
		case "llama3000":
			sp.Write([]byte("left\r"))
		default:
			sp.Write([]byte("right\r"))
		}
	default:
		// servo is reversed?
		sp.Write([]byte("right\r"))
	}
}
