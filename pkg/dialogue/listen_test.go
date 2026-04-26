package dialogue

import (
	"encoding/json"
	"testing"
	"time"
)

// mockMessage implements mqtt.Message for testing.
type mockMessage struct {
	payload []byte
	topic   string
}

func (m *mockMessage) Duplicate() bool   { return false }
func (m *mockMessage) Qos() byte         { return 0 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return m.topic }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}

func TestSomethingSaidJSONRoundTrip(t *testing.T) {
	original := SomethingSaid{Who: "alice", What: "hello world"}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded SomethingSaid
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Who != original.Who {
		t.Errorf("Who: got %q, want %q", decoded.Who, original.Who)
	}
	if decoded.What != original.What {
		t.Errorf("What: got %q, want %q", decoded.What, original.What)
	}
}

func TestSomethingSaidJSONFields(t *testing.T) {
	raw := `{"who":"bob","what":"testing"}`
	var s SomethingSaid
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if s.Who != "bob" {
		t.Errorf("Who: got %q, want %q", s.Who, "bob")
	}
	if s.What != "testing" {
		t.Errorf("What: got %q, want %q", s.What, "testing")
	}
}

func TestNewListenerEmptyVoices(t *testing.T) {
	_, err := NewListener("test", "tcp://localhost:1883", map[string]*Voice{})
	if err == nil {
		t.Fatal("expected error for empty voices map, got nil")
	}
}

func TestHandleSpeakingValidPayload(t *testing.T) {
	l := &Listener{
		whatWasSaid: make(chan SomethingSaid, 1),
		voices:      map[string]*Voice{},
	}

	payload, _ := json.Marshal(SomethingSaid{Who: "alice", What: "hello"})
	msg := &mockMessage{payload: payload, topic: "speak/alice"}

	l.handleSpeaking(nil, msg)

	select {
	case s := <-l.whatWasSaid:
		if s.Who != "alice" {
			t.Errorf("Who: got %q, want %q", s.Who, "alice")
		}
		if s.What != "hello" {
			t.Errorf("What: got %q, want %q", s.What, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message on channel")
	}
}

func TestHandleSpeakingInvalidPayload(t *testing.T) {
	l := &Listener{
		whatWasSaid: make(chan SomethingSaid, 1),
		voices:      map[string]*Voice{},
	}

	msg := &mockMessage{payload: []byte("not valid json"), topic: "speak/alice"}
	l.handleSpeaking(nil, msg)

	select {
	case <-l.whatWasSaid:
		t.Fatal("expected no message on channel for invalid JSON")
	default:
		// expected: nothing sent to channel
	}
}

func TestListenUnknownVoice(t *testing.T) {
	l := &Listener{
		whatWasSaid: make(chan SomethingSaid, 1),
		voices:      map[string]*Voice{},
	}

	l.whatWasSaid <- SomethingSaid{Who: "unknown", What: "hello"}
	close(l.whatWasSaid)

	// Listen drains the channel; reaching the end without panic is a pass.
	l.Listen()
}
