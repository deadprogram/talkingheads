package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/deadprogram/talkingheads/pkg/dialogue"
	"github.com/urfave/cli/v2"
)

var version = "dev"

func main() {
	app := &cli.App{
		Name:  "dialogue",
		Usage: "text-to-speech via configurable voices",
		Authors: []*cli.Author{
			{Name: "deadprogram"},
		},
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "connect to an MQTT server and process speak messages",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "server",
						Usage:    "MQTT server URL (e.g. tcp://localhost:1883)",
						Required: true,
						Aliases:  []string{"s"},
					},
					&cli.StringSliceFlag{
						Name:     "voice",
						Usage:    "voice specification in name:lang:model format, repeatable",
						Required: true,
						Aliases:  []string{"v"},
					},
					&cli.StringFlag{
						Name:    "data",
						Usage:   "data directory containing voice model files",
						Value:   "./voices",
						Aliases: []string{"d"},
					},
					&cli.BoolFlag{
						Name:  "gpu",
						Usage: "use GPU acceleration for TTS",
					},
				},
				Action: serveAction,
			},
			{
				Name:  "say",
				Usage: "say something once with a single voice and exit",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    "name of the voice speaker",
						Required: true,
						Aliases:  []string{"n"},
					},
					&cli.StringFlag{
						Name:     "lang",
						Usage:    "language code (e.g. en_US)",
						Required: true,
						Aliases:  []string{"l"},
					},
					&cli.StringFlag{
						Name:     "voice",
						Usage:    "voice model name",
						Required: true,
						Aliases:  []string{"v"},
					},
					&cli.StringFlag{
						Name:    "data",
						Usage:   "data directory containing voice model files",
						Value:   "./voices",
						Aliases: []string{"d"},
					},
					&cli.StringFlag{
						Name:     "say",
						Usage:    "text to speak",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "gpu",
						Usage: "use GPU acceleration for TTS",
					},
				},
				Action: sayAction,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func serveAction(c *cli.Context) error {
	server := c.String("server")
	dataDir := c.String("data")
	gpu := c.Bool("gpu")

	voices := make(map[string]*dialogue.Voice)
	for _, spec := range c.StringSlice("voice") {
		parts := strings.SplitN(spec, ":", 3)
		if len(parts) != 3 {
			return cli.Exit(fmt.Sprintf("invalid voice format %q: expected name:lang:model", spec), 1)
		}
		name, lang, model := parts[0], parts[1], parts[2]
		v, err := dialogue.NewVoice(name, lang, model, dataDir, gpu)
		if err != nil {
			return cli.Exit(fmt.Sprintf("failed to create voice %q: %v", name, err), 1)
		}
		voices[name] = v
	}

	listener, err := dialogue.NewListener("dialogue", server, voices)
	if err != nil {
		return cli.Exit(err, 1)
	}
	defer listener.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go listener.Listen()

	<-sigCh
	log.Println("shutting down")
	return nil
}

func sayAction(c *cli.Context) error {
	name := c.String("name")
	lang := c.String("lang")
	model := c.String("voice")
	dataDir := c.String("data")
	gpu := c.Bool("gpu")
	text := c.String("say")

	v, err := dialogue.NewVoice(name, lang, model, dataDir, gpu)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to create voice: %v", err), 1)
	}

	return v.SayOnce(text)
}
