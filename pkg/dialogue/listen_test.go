package dialogue

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/deadprogram/talkingheads/pkg/commands"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// mockToken is a no-op mqtt.Token used by mockMQTTClient.
type mockToken struct{}

func (t *mockToken) Wait() bool                       { return true }
func (t *mockToken) WaitTimeout(_ time.Duration) bool { return true }
func (t *mockToken) Done() <-chan struct{}            { ch := make(chan struct{}); close(ch); return ch }
func (t *mockToken) Error() error                     { return nil }

// publishedMessage records a single Publish call.
type publishedMessage struct {
	topic   string
	payload []byte
}

// mockMQTTClient captures Publish calls; all other mqtt.Client methods are no-ops.
type mockMQTTClient struct {
	mu       sync.Mutex
	messages []publishedMessage
}

func (c *mockMQTTClient) published() []publishedMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]publishedMessage, len(c.messages))
	copy(cp, c.messages)
	return cp
}

func (c *mockMQTTClient) IsConnected() bool      { return true }
func (c *mockMQTTClient) IsConnectionOpen() bool { return true }
func (c *mockMQTTClient) Connect() mqtt.Token    { return &mockToken{} }
func (c *mockMQTTClient) Disconnect(_ uint)      {}
func (c *mockMQTTClient) Subscribe(_ string, _ byte, _ mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *mockMQTTClient) SubscribeMultiple(_ map[string]byte, _ mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *mockMQTTClient) Unsubscribe(_ ...string) mqtt.Token       { return &mockToken{} }
func (c *mockMQTTClient) AddRoute(_ string, _ mqtt.MessageHandler) {}
func (c *mockMQTTClient) OptionsReader() mqtt.ClientOptionsReader  { return mqtt.ClientOptionsReader{} }

func (c *mockMQTTClient) Publish(topic string, _ byte, _ bool, payload interface{}) mqtt.Token {
	c.mu.Lock()
	defer c.mu.Unlock()
	var b []byte
	switch v := payload.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	}
	c.messages = append(c.messages, publishedMessage{topic: topic, payload: b})
	return &mockToken{}
}

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
	original := commands.Speak{Who: "alice", What: "hello world"}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded commands.Speak
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
	var s commands.Speak
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
	_, err := NewListener("test", "tcp://localhost:1883", map[string]*Voice{}, false)
	if err == nil {
		t.Fatal("expected error for empty voices map, got nil")
	}
}

func TestHandleSpeakingValidPayload(t *testing.T) {
	l := &Listener{
		whatWasSaid: make(chan commands.Speak, 1),
		voices:      map[string]*Voice{},
		verbose:     false,
	}

	payload, _ := json.Marshal(commands.Speak{Who: "alice", What: "hello"})
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
		whatWasSaid: make(chan commands.Speak, 1),
		voices:      map[string]*Voice{},
		verbose:     false,
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
		whatWasSaid: make(chan commands.Speak, 1),
		voices:      map[string]*Voice{},
		verbose:     false,
	}

	l.whatWasSaid <- commands.Speak{Who: "unknown", What: "hello"}
	close(l.whatWasSaid)

	// Listen drains the channel; reaching the end without panic is a pass.
	l.Listen()
}

func TestSayJSONRoundTrip(t *testing.T) {
	original := commands.Say{Who: "alice", What: "hello world"}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded commands.Say
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

func TestHandleSayValidPayload(t *testing.T) {
	l := &Listener{
		whatWasSaid: make(chan commands.Speak, 1),
		voices:      map[string]*Voice{},
		verbose:     false,
	}

	payload, _ := json.Marshal(commands.Say{Who: "alice", What: "hello"})
	msg := &mockMessage{payload: payload, topic: "say/alice"}

	l.handleSay(nil, msg)

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

func TestHandleSayInvalidPayload(t *testing.T) {
	l := &Listener{
		whatWasSaid: make(chan commands.Speak, 1),
		voices:      map[string]*Voice{},
		verbose:     false,
	}

	msg := &mockMessage{payload: []byte("not valid json"), topic: "say/alice"}
	l.handleSay(nil, msg)

	select {
	case <-l.whatWasSaid:
		t.Fatal("expected no message on channel for invalid JSON")
	default:
		// expected: nothing sent to channel
	}
}

func TestPublishSpeaking_Speaking(t *testing.T) {
	mc := &mockMQTTClient{}
	l := &Listener{client: mc, whatWasSaid: make(chan commands.Speak, 1), voices: map[string]*Voice{}, verbose: false}

	l.publishSpeaking("alice", commands.StatusSpeaking)

	msgs := mc.published()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(msgs))
	}
	if msgs[0].topic != "speaking/alice" {
		t.Errorf("topic: got %q, want %q", msgs[0].topic, "speaking/alice")
	}
	var s commands.Speaking
	if err := json.Unmarshal(msgs[0].payload, &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if s.Who != "alice" {
		t.Errorf("Who: got %q, want %q", s.Who, "alice")
	}
	if s.Status != commands.StatusSpeaking {
		t.Errorf("Status: got %q, want %q", s.Status, commands.StatusSpeaking)
	}
}

func TestPublishSpeaking_Stopped(t *testing.T) {
	mc := &mockMQTTClient{}
	l := &Listener{client: mc, whatWasSaid: make(chan commands.Speak, 1), voices: map[string]*Voice{}, verbose: false}

	l.publishSpeaking("bob", commands.StatusStopped)

	msgs := mc.published()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(msgs))
	}
	if msgs[0].topic != "speaking/bob" {
		t.Errorf("topic: got %q, want %q", msgs[0].topic, "speaking/bob")
	}
	var s commands.Speaking
	if err := json.Unmarshal(msgs[0].payload, &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if s.Who != "bob" {
		t.Errorf("Who: got %q, want %q", s.Who, "bob")
	}
	if s.Status != commands.StatusStopped {
		t.Errorf("Status: got %q, want %q", s.Status, commands.StatusStopped)
	}
}
