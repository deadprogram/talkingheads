package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/deadprogram/talkingheads/cmd/panelist/llm"
	"github.com/hybridgroup/go-sayanything/pkg/say"
	"github.com/hybridgroup/go-sayanything/pkg/tts"
	"github.com/tmc/langchaingo/llms"
	"github.com/urfave/cli/v2"
	"go.bug.st/serial"
)

var version = "dev"

var (
	sp                            serial.Port
	model, lang, name, human, led string
)

func main() {
	RunCLI(version)
}

// RunCLI runs the CLI command
func RunCLI(version string) error {
	app := &cli.App{
		Name:  "panelist",
		Usage: "stop making sense",
		Authors: []*cli.Author{
			{
				Name: "deadprogram",
			},
		},
		Flags: flagList,
		Action: func(c *cli.Context) error {
			model = c.String("model")
			lang = c.String("lang")
			voice := c.String("voice")
			keys := c.String("keys")
			port := c.String("port")
			name = c.String("name")
			human = c.String("human")
			led = c.String("led")

			if len(port) > 0 {
				sp, _ = serial.Open(port, &serial.Mode{BaudRate: 115200})
			}

			var t tts.Speaker
			var format string

			ttsengine := c.String("tts-engine")
			switch ttsengine {
			case "google":
				if keys == "" {
					return cli.Exit(errors.New("keyfile required. use -k=/path/to/keys.json"), 1)
				}

				t = tts.NewGoogle(lang, voice)
				if err := t.Connect(keys); err != nil {
					return cli.Exit(err, 1)
				}
				format = "mp3"
			case "piper":
				t = tts.NewPiper(lang, voice)
				if err := t.Connect(c.String("data")); err != nil {
					return cli.Exit(err, 1)
				}
				if c.Bool("gpu") {
					t.(*tts.Piper).UseGPU(true)
				}
				format = "wav"
			default:
				return cli.Exit(errors.New("tts-engine required. use -tts-engine=google or -tts-engine=piper"), 1)
			}

			defer t.Close()

			p := say.NewPlayer(format)
			defer p.Close()

			if c.String("speak") != "" {
				return SayAnythingOnce(t, p, c.String("speak"))
			}

			questions := make(chan llms.HumanChatMessage, 1)
			speaking := make(chan string, 1)
			others := make(chan llms.GenericChatMessage, 1)
			listening := make(chan string, 1)

			var replies chan llms.AIChatMessage
			if c.String("server") != "" {
				replies = make(chan llms.AIChatMessage, 1)
			}

			var seedPrompt, seedQuestion, seedResponse string
			switch model {
			case "llama3", "Lexi-Llama-3-8B-Uncensored_Q4_K_M":
				seedPrompt = llamaSeedPrompt
			case "gemma", "gemma2":
				seedPrompt = gemmaSeedPrompt
			case "phi3", "dolphin-2.9.2-Phi-3-Medium-abliterated-IQ4_XS", "Phi-3-mini-128k-instruct-abliterated-v3_q4":
				seedPrompt = phiSeedPrompt
			default:
				log.Fatal("failed creating LLM model: ", model)
			}

			llmConf := llm.Config{
				ModelName:       model,
				HistSize:        10,
				SeedPrompt:      seedPrompt,
				InitialQuestion: seedQuestion,
				InitialResponse: seedResponse,
			}
			l, err := llm.New(llmConf)
			if err != nil {
				log.Fatal("failed creating LLM client: ", err)
			}

			go l.Stream(context.Background(), questions, speaking, replies, others)
			go StartSayingAnything(t, p, speaking, listening)

			speaking <- name + " ready."

			if c.String("server") != "" {
				go startMQTT(name, c.String("server"), questions, speaking, replies, others, listening)

				select {
				case <-context.Background().Done():
					return nil
				}
			}

			return startKeyboardInput(questions)
		},
	}

	if err := app.Run(os.Args); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}
