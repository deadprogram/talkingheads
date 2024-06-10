package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func startKeyboardInput(questions chan question) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 {
			continue
		}

		var to, query string

		switch {
		case strings.HasPrefix(text, "llama:"):
			to = "llama"
			query = strings.TrimPrefix(text, "llama:")
		case strings.HasPrefix(text, "phi:"):
			to = "phi"
			query = strings.TrimPrefix(text, "phi:")
		case strings.HasPrefix(text, "gemma:"):
			to = "gemma"
			query = strings.TrimPrefix(text, "gemma:")
		default:
			fmt.Println("unknown recipient")
			continue
		}

		questions <- question{To: to, Content: query}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return nil
}
