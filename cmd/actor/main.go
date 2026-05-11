package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/deadprogram/talkingheads/pkg/actor"
	"github.com/hybridgroup/yzma/pkg/download"
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
				Name:    "libpath",
				Usage:   "path to the llama.cpp library directory",
				Aliases: []string{"l"},
			},
			&cli.StringFlag{
				Name:    "processor",
				Usage:   "processor to use (cpu or cuda)",
				Aliases: []string{"p"},
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "model-url",
				Usage:   "URL of the model to download and use (e.g. a HuggingFace URL)",
				Aliases: []string{"u"},
			},
			&cli.BoolFlag{
				Name:  "update-install",
				Usage: "update the installation of yzma if a new version of llama.cpp is available",
				Value: false,
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
				Usage:   "actor name used for MQTT topics direction/<name> and speak/<name>",
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
			&cli.StringFlag{
				Name:  "theme",
				Usage: "personality color sent to the action firmware on startup (red, green, blue, purple, orange, yellow)",
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
			&cli.Float64Flag{
				Name:  "min-p",
				Usage: "min-p sampling threshold (minimum probability relative to the most likely token; 0.0 = disabled)",
				Value: float64(actor.DefaultMinP),
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
			&cli.IntFlag{
				Name:  "batch-size",
				Usage: "logical maximum batch size (n_batch); 0 = use llama.cpp default",
				Value: int(actor.DefaultBatchSize),
			},
			&cli.IntFlag{
				Name:  "ubatch-size",
				Usage: "physical maximum micro-batch size (n_ubatch); 0 = use llama.cpp default",
				Value: int(actor.DefaultUBatchSize),
			},
			&cli.StringFlag{
				Name:  "model-format",
				Usage: "override the model format used for tool-call grammar (auto, standard, qwen, glm, mistral, gemma3, gemma, gpt, phi); default is auto-detect from model name",
				Value: "auto",
			},
			&cli.BoolFlag{
				Name:  "inject-tools",
				Usage: "enable injecting tool definitions into the system prompt (useful for models with native tool-call support)",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "enable-thinking",
				Usage: "enable thinking/reasoning mode for models that support it (e.g. Qwen3); disable to suppress chain-of-thought output",
				Value: false,
			},
			&cli.Float64Flag{
				Name:  "repeat-penalty",
				Usage: "penalise recently-seen tokens to reduce repetition (1.0 = disabled; try 1.1–1.3 for verbose models)",
				Value: float64(actor.DefaultRepeatPenalty),
			},
			&cli.Float64Flag{
				Name:  "freq-penalty",
				Usage: "penalise tokens proportional to how often they have appeared (0.0 = disabled)",
				Value: float64(actor.DefaultFreqPenalty),
			},
			&cli.Float64Flag{
				Name:  "presence-penalty",
				Usage: "penalise any token that has appeared at all (0.0 = disabled)",
				Value: float64(actor.DefaultPresencePenalty),
			},
			&cli.Float64Flag{
				Name:  "dry-multiplier",
				Usage: "DRY repetition penalty multiplier (0.0 = disabled; try 0.8 to curb looping)",
				Value: float64(actor.DefaultDryMultiplier),
			},
			&cli.IntFlag{
				Name:  "pause-interval",
				Usage: "seconds between repeated pause words while waiting for the model's first token (0 = use default)",
				Value: actor.DefaultPauseInterval,
			},
			&cli.IntFlag{
				Name:  "max-sentences",
				Usage: "maximum number of sentences spoken per turn; sentences beyond the limit are dropped (0 = unlimited)",
				Value: actor.DefaultMaxSentences,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "enable verbose logging for debugging",
				Value: false,
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	fmt.Print("\033[H\033[2J")
	name := c.String("name")
	fmt.Print(makeBanner(name))
	fmt.Println()

	modelURL := c.String("model-url")
	processor := c.String("processor")
	updateInstall := c.Bool("update-install")
	verbose := c.Bool("verbose")
	scriptFiles := c.StringSlice("script")
	server := c.String("server")
	serialPort := c.String("serial")
	baudRate := c.Int("baud")
	theme := strings.ToLower(strings.TrimSpace(c.String("theme")))

	if len(modelURL) == 0 {
		return cli.Exit("--model-url is required", 1)
	}

	libPath := c.String("libpath")
	if len(libPath) == 0 && os.Getenv("YZMA_LIB") != "" {
		libPath = os.Getenv("YZMA_LIB")
	}

	if len(libPath) == 0 {
		return cli.Exit("library path is required (set with --libpath or YZMA_LIB environment variable)", 1)
	}

	// Propagate the resolved library path to YZMA_LIB so downstream code
	// (e.g. actor.NewActor) loads llama.cpp from the same location.
	if err := os.Setenv("YZMA_LIB", libPath); err != nil {
		return cli.Exit(fmt.Sprintf("failed to set YZMA_LIB: %v", err), 1)
	}

	if err := actor.EnsureInstall(libPath, processor, updateInstall); err != nil {
		return cli.Exit(fmt.Sprintf("failed to install yzma: %v", err), 1)
	}

	if err := actor.EnsureModel(modelURL, download.DefaultModelsDir()); err != nil {
		return cli.Exit(fmt.Sprintf("failed to download model: %v", err), 1)
	}

	modelName := filepath.Base(modelURL)
	modelPath := filepath.Join(download.DefaultModelsDir(), modelName)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	if theme != "" {
		if commander == nil {
			commander = &actor.LogCommander{}
		}
		if err := commander.Send("theme " + theme); err != nil {
			if verbose {
				log.Printf("failed to set theme %q: %v", theme, err)
			}
		} else {
			if verbose {
				log.Printf("set theme to %s", theme)
			}
		}
	}

	var moreFunc func(*[]message.Message)
	var outputFunc func(string)

	cfg := actor.DefaultConfig()
	cfg.Temperature = float32(c.Float64("temperature"))
	cfg.TopP = float32(c.Float64("top-p"))
	cfg.MinP = float32(c.Float64("min-p"))
	cfg.TopK = int32(c.Int("top-k"))
	cfg.MaxTokens = c.Int("max-tokens")
	cfg.ContextSize = uint32(c.Int("context-size"))
	cfg.BatchSize = uint32(c.Int("batch-size"))
	cfg.UBatchSize = uint32(c.Int("ubatch-size"))
	cfg.ModelFormat = parseModelFormat(c.String("model-format"))
	cfg.InjectTools = c.Bool("inject-tools")
	cfg.EnableThinking = c.Bool("enable-thinking")
	cfg.RepeatPenalty = float32(c.Float64("repeat-penalty"))
	cfg.FreqPenalty = float32(c.Float64("freq-penalty"))
	cfg.PresencePenalty = float32(c.Float64("presence-penalty"))
	cfg.DryMultiplier = float32(c.Float64("dry-multiplier"))
	cfg.PauseInterval = c.Int("pause-interval")
	cfg.MaxSentences = c.Int("max-sentences")
	cfg.Verbose = verbose

	if server != "" {
		ml, err := actor.NewMQTTListener(name, server, commander, cfg.PauseWords, verbose)
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
		log.Printf("MQTT mode: listening on direction/%s, publishing to speak/%s\n", name, name)
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

	a, err := actor.NewActor(modelPath, cfg, commander, moreFunc, outputFunc)
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

// parseModelFormat converts a format name string to a message.Format value.
// Returns message.FormatAuto for unknown or empty strings so that
// auto-detection from the model path is used as the fallback.
func parseModelFormat(s string) message.Format {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "standard":
		return message.FormatStandard
	case "qwen":
		return message.FormatQwen
	case "glm":
		return message.FormatGLM
	case "mistral":
		return message.FormatMistral
	case "gemma3":
		return message.FormatGemma3
	case "gemma":
		return message.FormatGemma
	case "gpt":
		return message.FormatGPT
	case "phi":
		return message.FormatPhi
	default:
		return message.FormatAuto
	}
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

	return strings.TrimSpace(sb.String()), nil
}
