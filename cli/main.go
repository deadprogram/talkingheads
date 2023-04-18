package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
	//"time"

	"github.com/hybridgroup/go-sayanything/pkg/say"
	"github.com/hybridgroup/go-sayanything/pkg/tts"
	"github.com/urfave/cli/v2"
	"go.bug.st/serial"
)

var version = "dev"

var (
	sp   serial.Port
	lang string
)

func main() {
	RunCLI(version)
}

// RunCLI runs the CLI command
func RunCLI(version string) error {
	app := &cli.App{
		Name:      "talkinghead",
		Usage:     "just a head",
		UsageText: "talkinghead <TEXT_TO_SAY>\n   echo \"TEXT_TO_SAY\" | talkinghead",
		Authors: []*cli.Author{
			{
				Name: "deadprogram",
			},
		},
		Flags: []cli.Flag{
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
		},
		Before: func(c *cli.Context) error {
			if c.NArg() == 0 && !isPiped() {
				return cli.Exit("missing text to play", 1)
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			text := strings.Join(c.Args().Slice(), " ")

			lang = c.String("lang")
			voice := c.String("voice")
			keys := c.String("keys")
			port := c.String("port")
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

			// input piped to stdin
			if isPiped() {
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					err := SayAnything(t, p, scanner.Text())
					if err != nil {
						return cli.Exit(err, 1)
					}
				}

				if err := scanner.Err(); err != nil {
					return cli.Exit(err, 1)
				}
				return nil
			}

			return SayAnything(t, p, text)
		},
	}

	if err := app.Run(os.Args); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func SayAnything(t *tts.Google, p *say.Player, text string) error {
	if len(text) == 0 {
		return nil
	}

	if lang == "es-ES" && strings.Contains(text, "### Human:") {
		// error?
		return nil
	}

	text = cleanupText(text, "### Human:")
	text = cleanupText(text, "### Assistant:")
	text = cleanupText(text, "### Humano:")
	text = cleanupText(text, "### Asistente:")
	text = cleanupText(text, "`")

	if strings.HasPrefix(text, ">") {
		text = strings.TrimPrefix(text, ">")
	}

	data, err := t.Speech(text)
	if err != nil {
		return err
	}

	if sp != nil {
		sp.Write([]byte("talk\r"))
		//time.Sleep(100 * time.Millisecond)
		defer sp.Write([]byte("stop\r"))
	}

	return p.Say(data)
}

func isPiped() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	notPipe := info.Mode()&os.ModeNamedPipe == 0
	return !notPipe
}

func cleanupText(text, cleanup string) string {
	if strings.Contains(text, cleanup) {
		return strings.ReplaceAll(text, cleanup, "")
	}

	return text
}
