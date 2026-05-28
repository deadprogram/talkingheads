package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPauseWords_BasicPhrases(t *testing.T) {
	f := writeTempFile(t, "let me think...\ngive me a moment...\nhold on...\n")
	words, err := loadPauseWords(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"let me think...", "give me a moment...", "hold on..."}
	if len(words) != len(want) {
		t.Fatalf("got %d words, want %d: %v", len(words), len(want), words)
	}
	for i, w := range want {
		if words[i] != w {
			t.Errorf("[%d] got %q, want %q", i, words[i], w)
		}
	}
}

func TestLoadPauseWords_SkipsBlankLines(t *testing.T) {
	f := writeTempFile(t, "first phrase...\n\n\nsecond phrase...\n")
	words, err := loadPauseWords(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 2 {
		t.Fatalf("got %d words, want 2: %v", len(words), words)
	}
}

func TestLoadPauseWords_SkipsCommentLines(t *testing.T) {
	f := writeTempFile(t, "# this is a comment\nreal phrase...\n# another comment\n")
	words, err := loadPauseWords(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 1 || words[0] != "real phrase..." {
		t.Fatalf("got %v, want [real phrase...]", words)
	}
}

func TestLoadPauseWords_TrimsWhitespace(t *testing.T) {
	f := writeTempFile(t, "  leading spaces...\ntrailing spaces...  \n  both ends...  \n")
	words, err := loadPauseWords(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"leading spaces...", "trailing spaces...", "both ends..."}
	for i, w := range want {
		if words[i] != w {
			t.Errorf("[%d] got %q, want %q", i, words[i], w)
		}
	}
}

func TestLoadPauseWords_EmptyFile(t *testing.T) {
	f := writeTempFile(t, "")
	words, err := loadPauseWords(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 0 {
		t.Errorf("expected empty slice, got %v", words)
	}
}

func TestLoadPauseWords_FileNotFound(t *testing.T) {
	_, err := loadPauseWords("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "pause_words.txt")
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return f
}
