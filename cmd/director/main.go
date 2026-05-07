package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
)

var version = "dev"

var actors = []string{}

// actorAliases maps a canonical actor name to a list of alternative spoken
// names (e.g. whisper.cpp mis-transcriptions). Populated via
// --hotmic-actor-alias flags.
var actorAliases = map[string][]string{}

// fuzzyThreshold is the maximum allowed Levenshtein distance ratio (0–1) for
// actor name matching. Configurable via --hotmic-fuzzy-threshold.
var fuzzyThreshold = 0.6

func main() {
	if err := RunCLI(version); err != nil {
		log.Fatal(err)
	}
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
			&cli.StringSliceFlag{
				Name:  "actor",
				Usage: "canonical actor name (repeatable); e.g. --actor llama3000 --actor gemmai",
			},
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
			&cli.Float64Flag{
				Name:  "hotmic-fuzzy-threshold",
				Usage: "maximum Levenshtein distance ratio (0–1) for fuzzy actor name matching; lower values require a closer match",
				Value: 0.6,
			},
			&cli.StringSliceFlag{
				Name:  "hotmic-actor-alias",
				Usage: "map alternate spoken names to a canonical actor: --hotmic-actor-alias gemmai:jami|jamai|jenna (repeatable)",
			},
		},
		Action: func(c *cli.Context) error {
			fmt.Print("\033[H\033[2J")
			fmt.Print(banner)
			fmt.Println()

			if c.String("server") == "" {
				log.Fatal("server is required")
			}

			actors = c.StringSlice("actor")
			if len(actors) == 0 {
				log.Fatal("at least one --actor is required")
			}
			conv, err := newConversation(c.String("server"))
			if err != nil {
				log.Fatal(err)
			}

			go conv.processQuestions()

			for _, entry := range c.StringSlice("hotmic-actor-alias") {
				parts := strings.SplitN(entry, ":", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return fmt.Errorf("invalid --hotmic-actor-alias %q: expected name:alias1|alias2|...", entry)
				}
				name := parts[0]
				for _, alt := range strings.Split(parts[1], "|") {
					if alt = strings.TrimSpace(alt); alt != "" {
						actorAliases[name] = append(actorAliases[name], alt)
					}
				}
			}

			if modelPath := c.String("hotmic-model"); modelPath != "" {
				key := rune(c.String("hotmic-key")[0])
				fuzzyThreshold = c.Float64("hotmic-fuzzy-threshold")
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
