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

		kind := kindDirection
		content := strings.TrimSpace(query)
		if rest, ok := stripSayPrefix(content); ok {
			kind = kindSay
			content = trimSurroundingQuotes(rest)
		}

		questions <- question{To: to, Content: content, Kind: kind}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func displayQuestion() {
	fmt.Printf("Enter a question for an actor %v:\n", actors)
}

// stripSayPrefix returns the remainder of s with the leading "say" word
// removed (case-insensitive) and reports whether the prefix was present.
// Trailing punctuation directly after "say" (e.g. "say:" or "say,") is
// tolerated.
func stripSayPrefix(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return s, false
	}
	space := strings.IndexAny(trimmed, " \t")
	var word, rest string
	if space < 0 {
		word = trimmed
		rest = ""
	} else {
		word = trimmed[:space]
		rest = trimmed[space+1:]
	}
	word = strings.TrimRight(word, ":,")
	if !strings.EqualFold(word, "say") {
		return s, false
	}
	return strings.TrimSpace(rest), true
}
