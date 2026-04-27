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
			&cli.StringFlag{
				Name:    "server",
				Usage:   "MQTT broker URL (e.g. tcp://localhost:1883); enables MQTT mode",
				Aliases: []string{"b"},
			},
			&cli.StringFlag{
				Name:    "name",
				Usage:   "actor name used for MQTT topics ask/<name> and speak/<name>",
				Value:   "actor",
				Aliases: []string{"n"},
			},
			&cli.StringFlag{
				Name:  "serial",
				Usage: "serial port to send action commands to (e.g. /dev/ttyACM0); if omitted, commands are logged to console",
			},
			&cli.IntFlag{
				Name:  "baud",
				Usage: "baud rate for the serial port",
				Value: 9600,
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
	server := c.String("server")
	name := c.String("name")
	serialPort := c.String("serial")
	baudRate := c.Int("baud")

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

	var moreFunc func(*[]model.D)
	var outputFunc func(string)

	if server != "" {
		ml, err := actor.NewMQTTListener(name, server)
		if err != nil {
			return cli.Exit(fmt.Sprintf("failed to connect to MQTT broker: %v", err), 1)
		}
		defer ml.Close()
		// Unblock MoreFunc when the context is cancelled.
		go func() {
			<-ctx.Done()
			ml.Close()
		}()
		moreFunc = ml.MoreFunc()
		outputFunc = ml.OutputFunc()
		log.Printf("MQTT mode: listening on ask/%s, publishing to speak/%s\n", name, name)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		moreFunc = func(conversation *[]model.D) {
			fmt.Print("\nYou: ")
			if !scanner.Scan() {
				stop()
				return
			}
			line := scanner.Text()
			if line == "" {
				return
			}
			*conversation = append(*conversation, model.D{"role": "user", "content": line})
		}
		outputFunc = func(content string) {
			fmt.Printf("\nActor: %s\n", content)
		}
	}

	var commander actor.Commander
	if serialPort != "" {
		sc, err := actor.NewSerialCommander(serialPort, baudRate)
		if err != nil {
			return cli.Exit(fmt.Sprintf("failed to open serial port: %v", err), 1)
		}
		defer sc.Close()
		commander = sc
		log.Printf("Serial mode: sending action commands to %s at %d baud\n", serialPort, baudRate)
	}

	a, err := actor.NewActor(mp, commander, moreFunc, outputFunc)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to create actor: %v", err), 1)
	}

	fmt.Println("Actor ready. Type your message and press Enter. Use Ctrl+C or Ctrl+D to quit.")

	if err := a.Run(ctx, systemPrompt); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}
