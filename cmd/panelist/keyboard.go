package main

import (
	"bufio"
	"log"
	"os"
	"time"

	"github.com/tmc/langchaingo/llms"
)

func startKeyboardInput(questions chan llms.HumanChatMessage) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		question := scanner.Text()
		if len(question) == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		questions <- llms.HumanChatMessage{Content: question}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return nil
}
