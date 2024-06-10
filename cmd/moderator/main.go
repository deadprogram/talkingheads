package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

var version = "dev"

var (
	model, lang, name, human, led string
)

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
		},
		Action: func(c *cli.Context) error {
			if c.String("server") == "" {
				log.Fatal("server is required")
			}

			conv, err := startConversation(c.String("server"))
			if err != nil {
				log.Fatal(err)
			}

			return startKeyboardInput(conv.questions)
		},
	}

	if err := app.Run(os.Args); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}
