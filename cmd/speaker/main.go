package main

import (
	"bufio"
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/deadprogram/talkingheads/cmd/speaker/llm"
	"github.com/hybridgroup/go-sayanything/pkg/say"
	"github.com/hybridgroup/go-sayanything/pkg/tts"
	"github.com/urfave/cli/v2"
	"go.bug.st/serial"
)

var version = "dev"

var (
	sp                                 serial.Port
	model, lang, assistant, human, led string
)

func main() {
	RunCLI(version)
}

// RunCLI runs the CLI command
func RunCLI(version string) error {
	app := &cli.App{
		Name:      "talkingheads",
		Usage:     "stop making sense",
		UsageText: "talkingheads <TEXT_TO_SAY>\n   echo \"TEXT_TO_SAY\" | talkingheads",
		Authors: []*cli.Author{
			{
				Name: "deadprogram",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "model",
				Usage:   "model to use",
				Value:   "llama2",
				Aliases: []string{"m"},
			},
			&cli.StringFlag{
				Name:    "lang",
				Usage:   "language of the text",
				Value:   "en-us",
				Aliases: []string{"l"},
			},
			&cli.StringFlag{
				Name:  "voice",
				Usage: "voice to use to speak",
				Value: "",
			},
			&cli.StringFlag{
				Name:    "keys",
				Usage:   "Google TTS keyfile",
				Value:   "",
				Aliases: []string{"k"},
			},
			&cli.StringFlag{
				Name:    "port",
				Usage:   "port for LEDs",
				Value:   "",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "assistant",
				Usage:   "name of assistant",
				Value:   "Assistant",
				Aliases: []string{"a"},
			},
			&cli.StringFlag{
				Name:    "human",
				Usage:   "name of human",
				Value:   "Human",
				Aliases: []string{"hu"},
			},
			&cli.StringFlag{
				Name:  "led",
				Usage: "name led command",
				Value: "talk",
			},
			&cli.StringFlag{
				Name:  "speak",
				Usage: "just say something",
			},
		},
		Action: func(c *cli.Context) error {
			model = c.String("model")
			lang = c.String("lang")
			voice := c.String("voice")
			keys := c.String("keys")
			port := c.String("port")
			assistant = c.String("assistant")
			human = c.String("human")
			led = c.String("led")

			if len(port) > 0 {
				// open serial port
				sp, _ = serial.Open(port, &serial.Mode{BaudRate: 115200})
			}

			if keys == "" {
				return cli.Exit(errors.New("keyfile required. use -k=/path/to/keys.json"), 1)
			}

			t := tts.NewGoogle(lang, voice)
			if err := t.Connect(keys); err != nil {
				return cli.Exit(err, 1)
			}

			defer t.Close()

			p := say.NewPlayer()
			defer p.Close()

			if c.String("speak") != "" {
				return SayAnythingOnce(t, p, c.String("speak"))
			}

			prompts := make(chan string)
			replies := make(chan string)

			var seedPrompt, seedQuestion, seedResponse string
			switch model {
			case "llama2":
				seedPrompt = llamaSeedPrompt
				seedQuestion = llamaQuestionPrompt
				seedResponse = llamaResponsePrompt
			case "gemma":
				seedPrompt = gemmaSeedPrompt
				seedQuestion = gemmaQuestionPrompt
				seedResponse = gemmaResponsePrompt
			case "phi3":
				seedPrompt = phiSeedPrompt
				seedQuestion = phiQuestionPrompt
				seedResponse = phiResponsePrompt
			default:
				seedPrompt = llamaSeedPrompt
				seedQuestion = llamaQuestionPrompt
				seedResponse = llamaResponsePrompt
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

			go l.Stream(context.Background(), prompts, replies)
			go StartSayingAnything(t, p, replies)
			replies <- "ok ready"

			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				prompt := scanner.Text()
				if len(prompt) == 0 {
					continue
				}

				prompts <- prompt
			}

			if err := scanner.Err(); err != nil {
				return cli.Exit(err, 1)
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func StartSayingAnything(t *tts.Google, p *say.Player, responses chan string) error {
	for text := range responses {
		err := SayAnything(t, p, text)
		if err != nil {
			return err
		}
	}

	return nil
}

var speaking = 0

func SayAnything(t *tts.Google, p *say.Player, text string) error {
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

func SayAnythingOnce(t *tts.Google, p *say.Player, text string) error {
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

	p.Say(data)
	speaking--

	if sp != nil {
		if speaking == 0 {
			sp.Write([]byte("stop\r"))
		}
	}

	return nil
}

func cleanupText(text, cleanup string) string {
	if strings.Contains(text, cleanup) {
		return strings.ReplaceAll(text, cleanup, "")
	}

	return text
}
