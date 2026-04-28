package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/deadprogram/talkingheads/pkg/actor"
	"github.com/hybridgroup/yzma/pkg/message"
	"github.com/urfave/cli/v2"
)

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
			&cli.StringSliceFlag{
				Name:    "script",
				Usage:   "path to a system prompt file (repeatable; files are concatenated in order)",
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
			&cli.Float64Flag{
				Name:  "temperature",
				Usage: "sampling temperature",
				Value: float64(actor.DefaultTemperature),
			},
			&cli.Float64Flag{
				Name:  "top-p",
				Usage: "top-p (nucleus) sampling threshold",
				Value: float64(actor.DefaultTopP),
			},
			&cli.IntFlag{
				Name:  "top-k",
				Usage: "top-k sampling limit",
				Value: int(actor.DefaultTopK),
			},
			&cli.IntFlag{
				Name:  "max-tokens",
				Usage: "maximum number of tokens to generate per turn",
				Value: actor.DefaultMaxTokens,
			},
			&cli.IntFlag{
				Name:  "context-size",
				Usage: "KV cache / context window size in tokens",
				Value: int(actor.DefaultContextSize),
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
	scriptFiles := c.StringSlice("script")
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

	var err error

	if modelURL != "" {
		log.Println("downloading model, this may take a while...")
		modelPath, err = actor.InstallSystem(modelURL)
		if err != nil {
			return cli.Exit(fmt.Sprintf("failed to install model: %v", err), 1)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var moreFunc func(*[]message.Message)
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
		moreFunc = func(conversation *[]message.Message) {
			fmt.Print("\nYou: ")
			if !scanner.Scan() {
				stop()
				return
			}
			line := scanner.Text()
			if line == "" {
				return
			}
			*conversation = append(*conversation, message.Chat{Role: "user", Content: line})
		}
		outputFunc = func(content string) {
			fmt.Printf("\nActor: %s\n", content)
		}
	}

	systemPrompt, err := buildSystemPrompt(scriptFiles)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to load script: %v", err), 1)
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

	a, err := actor.NewActor(modelPath, actor.Config{
		Temperature: float32(c.Float64("temperature")),
		TopP:        float32(c.Float64("top-p")),
		TopK:        int32(c.Int("top-k")),
		MaxTokens:   c.Int("max-tokens"),
		ContextSize: uint32(c.Int("context-size")),
	}, commander, moreFunc, outputFunc)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to create actor: %v", err), 1)
	}
	defer a.Close()

	fmt.Println("Actor ready. Use Ctrl+C or Ctrl+D to quit.")

	if err := a.Run(ctx, systemPrompt); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}

// buildSystemPrompt reads each file in paths and concatenates their contents,
// separated by a blank line. If no files are provided, a sensible default is
// returned so the actor is always usable without a script.
func buildSystemPrompt(paths []string) (string, error) {
	if len(paths) == 0 {
		return "You are a helpful assistant.", nil
	}

	var sb strings.Builder
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading script %q: %w", path, err)
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.Write(data)
	}

	fmt.Println(strings.TrimSpace(sb.String()))

	return strings.TrimSpace(sb.String()), nil
}
