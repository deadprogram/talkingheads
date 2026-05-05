package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/deadprogram/talkingheads/pkg/hotmic"
)

func startHotMicInput(ctx context.Context, questions chan question, modelPath, lang string, key rune) error {
	mic, err := hotmic.New(hotmic.Options{
		Key:       key,
		ModelPath: modelPath,
		Language:  lang,
	})
	if err != nil {
		return err
	}
	defer mic.Close()

	texts, err := mic.Listen(ctx)
	if err != nil {
		return err
	}

	// \r\n required: terminal is in raw mode at this point.
	fmt.Printf("hotmic ready — press the toggle key to record, speak \"<actor>: <question>\", press again to transcribe\r\n")

	for text := range texts {
		if text == "" {
			continue
		}

		// Split on the first punctuation that separates the actor name from
		// the question body. Whisper may transcribe the separator as ':', ',',
		// '?', or '.'.
		idx := strings.IndexAny(text, ":,?.")
		if idx < 0 {
			fmt.Printf("\rhotmic: no actor separator in %q\r\n", text)
			continue
		}
		nameRaw := strings.TrimSpace(text[:idx])
		content := strings.TrimSpace(text[idx+1:])

		to, ok := matchActor(nameRaw)
		if !ok {
			fmt.Printf("\rhotmic: unknown actor %q in %q — expected one of %v\r\n", nameRaw, text, actors)
			continue
		}

		fmt.Printf("\rhotmic: got question for %s: %q\r\n", to, content)

		select {
		case questions <- question{To: to, Content: content}:
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

// matchActor finds the actor whose name best matches spoken.
// It normalises both sides (lowercase, alphanumeric only) and applies three
// strategies in order:
//  1. Exact match after normalisation.
//  2. Substring containment (handles e.g. "lama" → "llama3000").
//  3. Fuzzy edit-distance: accept the closest actor when the Levenshtein
//     distance is ≤ 60 % of the longer normalised name (handles transcription
//     errors like "lamma3000"→"llama3000" or "jamai"→"gemmai").
func matchActor(spoken string) (string, bool) {
	norm := normalise(spoken)
	// 1. Exact match after normalisation.
	for _, p := range actors {
		if normalise(p) == norm {
			return p, true
		}
	}
	// 2. Substring fallback.
	for _, p := range actors {
		np := normalise(p)
		if strings.Contains(np, norm) || strings.Contains(norm, np) {
			return p, true
		}
	}
	// 3. Fuzzy fallback: pick the actor with the smallest edit distance.
	best, bestDist := "", int(^uint(0)>>1)
	for _, p := range actors {
		if d := levenshtein(norm, normalise(p)); d < bestDist {
			bestDist = d
			best = p
		}
	}
	if best != "" {
		maxLen := len(norm)
		if n := len(normalise(best)); n > maxLen {
			maxLen = n
		}
		if float64(bestDist) <= float64(maxLen)*0.6 {
			return best, true
		}
	}
	return "", false
}

// levenshtein returns the edit distance between a and b.
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ra := range a {
		curr[0] = i + 1
		for j, rb := range b {
			if ra == rb {
				curr[j+1] = prev[j]
			} else {
				curr[j+1] = 1 + minInt(prev[j+1], curr[j], prev[j])
			}
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minInt(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// normalise lowercases s and strips everything that is not a letter or digit.
func normalise(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
