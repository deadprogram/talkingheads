package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ardanlabs/kronk/sdk/kronk/model"
	"github.com/ardanlabs/kronk/sdk/tools/models"
	"github.com/deadprogram/talkingheads/pkg/actor"
	"github.com/urfave/cli/v2"
)

const defaultSystemPrompt = "You are a helpful assistant."

func main() {
	app := &cli.App{
		Name:  "actor",
		Usage: "have a conversation with an AI actor",
		Authors: []*cli.Author{
			{Name: "deadprogram"},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "model-url",
				Usage:   "URL of the model to download and use (e.g. a HuggingFace URL)",
				Aliases: []string{"u"},
			},
			&cli.StringFlag{
				Name:    "model-path",
				Usage:   "local path to a pre-downloaded model file",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "system-prompt",
				Usage:   "system prompt to set the actor's persona",
				Value:   defaultSystemPrompt,
				Aliases: []string{"s"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	modelURL := c.String("model-url")
	modelPath := c.String("model-path")
	systemPrompt := c.String("system-prompt")

	if modelURL == "" && modelPath == "" {
		return cli.Exit("one of --model-url or --model-path is required", 1)
	}
	if modelURL != "" && modelPath != "" {
		return cli.Exit("--model-url and --model-path are mutually exclusive", 1)
	}

	var mp models.Path
	var err error

	if modelURL != "" {
		log.Println("downloading model, this may take a while...")
		mp, err = actor.InstallSystem(modelURL)
		if err != nil {
			return cli.Exit(fmt.Sprintf("failed to install model: %v", err), 1)
		}
	} else {
		mp = models.Path{ModelFiles: []string{modelPath}}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	scanner := bufio.NewScanner(os.Stdin)

	moreFunc := func(conversation *[]model.D) {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			// EOF or signal — signal the actor to stop by returning without appending.
			stop()
			return
		}
		line := scanner.Text()
		if line == "" {
			return
		}
		*conversation = append(*conversation, model.D{"role": "user", "content": line})
	}

	outputFunc := func(content string) {
		fmt.Printf("\nActor: %s\n", content)
	}

	a, err := actor.NewActor(mp, moreFunc, outputFunc)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to create actor: %v", err), 1)
	}

	fmt.Println("Actor ready. Type your message and press Enter. Use Ctrl+C or Ctrl+D to quit.")

	if err := a.Run(ctx, systemPrompt); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}
