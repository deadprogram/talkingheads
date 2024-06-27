package main

import "github.com/urfave/cli/v2"

var flagList = []cli.Flag{
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
		Name:  "name",
		Usage: "name of assistant",
		Value: "Assistant",
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
	&cli.StringFlag{
		Name:  "server",
		Usage: "mqtt server",
	},
	&cli.StringFlag{
		Name:  "tts-engine",
		Usage: "text to speech engine",
	},
	&cli.BoolFlag{
		Name:  "gpu",
		Usage: "use GPU for TTS engine",
	},
	&cli.StringFlag{
		Name:  "data",
		Usage: "data directory for TTS engine",
	},
}
