package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
)

var version = "dev"

func main() {
	RunCLI(version)
}

// RunCLI runs the CLI command
func RunCLI(version string) error {
	app := &cli.App{
		Name:      "moderator",
		Usage:     "stop making sense",
		UsageText: "moderator",
		Authors: []*cli.Author{
			{
				Name: "deadprogram",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server",
				Usage: "mqtt server",
			},
			&cli.StringFlag{
				Name:  "hotmic-model",
				Usage: "path to a whisper.cpp GGML model file; enables hotmic input when set",
			},
			&cli.StringFlag{
				Name:  "hotmic-lang",
				Usage: "BCP-47 language code for hotmic transcription (e.g. \"en\"), or \"auto\"",
				Value: "auto",
			},
			&cli.StringFlag{
				Name:  "hotmic-key",
				Usage: "keyboard character that toggles hotmic recording on/off",
				Value: " ",
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("server") == "" {
				log.Fatal("server is required")
			}

			conv, err := newConversation(c.String("server"))
			if err != nil {
				log.Fatal(err)
			}

			go conv.processQuestions()

			if modelPath := c.String("hotmic-model"); modelPath != "" {
				key := rune(c.String("hotmic-key")[0])
				ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
				defer cancel()
				return startHotMicInput(ctx, conv.questions, modelPath, c.String("hotmic-lang"), key)
			}

			return startKeyboardInput(conv.questions)
		},
	}

	if err := app.Run(os.Args); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}
