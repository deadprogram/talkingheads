package main

import (
	"bufio"
	"log"
	"os"

	"github.com/tmc/langchaingo/llms"
)

func startKeyboardInput(questions chan llms.HumanChatMessage) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		question := scanner.Text()
		if len(question) == 0 {
			continue
		}

		questions <- llms.HumanChatMessage{Content: question}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return nil
}
