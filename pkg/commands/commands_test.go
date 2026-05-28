package commands

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSpeak_Thinking_OmittedWhenFalse(t *testing.T) {
	s := Speak{Who: "gemmai", What: "hello"}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(b), "thinking") {
		t.Errorf("expected 'thinking' to be omitted when false, got %s", b)
	}
}

func TestSpeak_Thinking_IncludedWhenTrue(t *testing.T) {
	s := Speak{Who: "gemmai", What: "let me think...", Thinking: true}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(b), `"thinking":true`) {
		t.Errorf("expected 'thinking':true in JSON, got %s", b)
	}
}

func TestSpeak_Thinking_RoundTrip(t *testing.T) {
	original := Speak{Who: "phineas", What: "give me a moment...", Thinking: true}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded Speak
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.Who != original.Who || decoded.What != original.What || decoded.Thinking != original.Thinking {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestSpeak_Thinking_FalseWhenAbsentInJSON(t *testing.T) {
	b := []byte(`{"who":"phineas","what":"The sky is blue."}`)
	var s Speak
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if s.Thinking {
		t.Errorf("expected Thinking to be false when absent in JSON, got true")
	}
}
