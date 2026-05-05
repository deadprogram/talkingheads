package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"
)

func startKeyboardInput(questions chan question) error {
	displayQuestion()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		var query string
		first := strings.Split(text, " ")[0]
		to := strings.TrimSuffix(first, ":")
		to = strings.TrimSuffix(to, ",")
		to = strings.ToLower(to)

		switch {
		case slices.Contains(actors, to):
			query = strings.TrimPrefix(text, first)
			displayQuestion()
		default:
			fmt.Println("unknown actor. try again:", actors)
			continue
		}

		questions <- question{To: to, Content: query}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func displayQuestion() {
	fmt.Printf("Enter a question for an actor %v:\n", actors)
}
