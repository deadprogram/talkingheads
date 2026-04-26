module github.com/deadprogram/talkingheads/cmd

go 1.26.0

replace github.com/tmc/langchaingo => github.com/treywelsh/langchaingo v0.0.0-20241010141243-207810224be9

require (
	github.com/deadprogram/talkingheads v0.0.0
	github.com/eclipse/paho.mqtt.golang v1.4.3
	github.com/urfave/cli/v2 v2.25.1
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20260420051257-fc674574ca27 // indirect
	github.com/gordonklaus/portaudio v0.0.0-20260203164431-765aa7dfa631 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/term v0.42.0 // indirect
)

replace github.com/deadprogram/talkingheads => /home/ron/Development/talkingheads

replace github.com/ggerganov/whisper.cpp/bindings/go => ../lib/whisper.cpp/bindings/go
