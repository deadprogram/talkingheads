// Package banner generates ASCII-art banners in the same "#" block
// style used by the actor, dialogue, and director commands.
package banner

import (
	"strings"

	figure "github.com/common-nighthawk/go-figure"
)

// Generate returns an ASCII-art banner for the given text using the
// FIGlet "banner" font, matching the style of the hard-coded banners
// in the cmd/* programs. The returned string starts and ends with a
// newline so it can be printed directly.
func Generate(text string) string {
	fig := figure.NewFigure(text, "banner", true)
	return "\n" + fig.String() + "\n"
}

// GenerateWithFont is like Generate but lets the caller pick any font
// supported by go-figure (e.g. "standard", "big", "small", "banner").
func GenerateWithFont(text, font string) string {
	fig := figure.NewFigure(text, font, true)
	return "\n" + fig.String() + "\n"
}

// TrimRightLines removes trailing whitespace from each line. Useful
// when embedding the banner into contexts that flag trailing spaces.
func TrimRightLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.Join(lines, "\n")
}
