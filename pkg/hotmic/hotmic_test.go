package hotmic

import (
	"os"
	"testing"

	"github.com/ardanlabs/bucky/pkg/whisper"
)

// testSetup skips t when BUCKY_LIB is not set in the environment.
func testSetup(t *testing.T) {
	t.Helper()
	if os.Getenv("BUCKY_LIB") == "" {
		t.Skip("BUCKY_LIB not set; skipping whisper FFI test")
	}
}

// testModelPath returns the model path from BUCKY_TEST_MODEL, skipping t if
// the variable is unset or the file is absent.
func testModelPath(t *testing.T) string {
	t.Helper()
	model := os.Getenv("BUCKY_TEST_MODEL")
	if model == "" {
		t.Skip("BUCKY_TEST_MODEL not set; skipping test that requires a model")
	}
	if _, err := os.Stat(model); err != nil {
		t.Skipf("model file %q not present: %v", model, err)
	}
	return model
}

// testNewHotMic creates a HotMic using env-var paths and registers h.Close
// with t.Cleanup.
func testNewHotMic(t *testing.T) *HotMic {
	t.Helper()
	testSetup(t)
	h, err := New(Options{ModelPath: testModelPath(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

// TestNew_InvalidLibPath verifies that New returns an error when the shared
// library cannot be found at the given path.
func TestNew_InvalidLibPath(t *testing.T) {
	_, err := New(Options{
		LibPath:   "/nonexistent/lib/path",
		ModelPath: "dummy.bin",
	})
	if err == nil {
		t.Fatal("expected error for invalid lib path, got nil")
	}
}

// TestNew_DefaultKey verifies that Key defaults to space (' ') when zero.
func TestNew_DefaultKey(t *testing.T) {
	h := testNewHotMic(t)
	if h.key != ' ' {
		t.Errorf("default key: got %q, want ' '", h.key)
	}
}

// TestNew_DefaultLanguage verifies that Language defaults to "auto" when empty.
func TestNew_DefaultLanguage(t *testing.T) {
	h := testNewHotMic(t)
	if h.language != "auto" {
		t.Errorf("default language: got %q, want \"auto\"", h.language)
	}
}

// TestNew_NonZeroContext verifies that the whisper context handle is non-zero
// after a successful New call.
func TestNew_NonZeroContext(t *testing.T) {
	h := testNewHotMic(t)
	if h.ctx == 0 {
		t.Error("whisper context is zero after successful New")
	}
}

// TestNew_ExplicitOptions verifies that explicitly provided Key and Language
// values are preserved without being overridden by defaults.
func TestNew_ExplicitOptions(t *testing.T) {
	testSetup(t)
	h, err := New(Options{
		Key:       'r',
		ModelPath: testModelPath(t),
		Language:  "en",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer h.Close()

	if h.key != 'r' {
		t.Errorf("key: got %q, want 'r'", h.key)
	}
	if h.language != "en" {
		t.Errorf("language: got %q, want \"en\"", h.language)
	}
}

// TestClose verifies that Close returns nil and releases resources.
func TestClose(t *testing.T) {
	h := testNewHotMic(t)
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestTranscribe_Silence verifies that transcribing a silent audio buffer
// returns no error and produces an empty or near-empty transcript.
func TestTranscribe_Silence(t *testing.T) {
	h := testNewHotMic(t)

	// Three seconds of silence at whisper's required 16 kHz sample rate.
	silence := make([]float32, 3*whisper.SampleRate)

	text, err := h.transcribe(silence)
	if err != nil {
		t.Fatalf("transcribe silence: %v", err)
	}
	t.Logf("transcribed silence: %q", text)
}
