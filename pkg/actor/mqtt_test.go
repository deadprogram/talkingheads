package actor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/deadprogram/talkingheads/pkg/commands"
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

// captureStdout runs f and returns everything written to os.Stdout during the call.
func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestHandleSpeakingStatus_Speaking(t *testing.T) {
	l := &MQTTListener{name: "testactor"}

	payload, _ := json.Marshal(commands.Speaking{Who: "testactor", Status: commands.StatusSpeaking})
	msg := &mockMessage{payload: payload, topic: "speaking/testactor"}

	output := captureStdout(func() {
		l.handleSpeakingStatus(nil, msg)
	})

	want := fmt.Sprintln("now speaking")
	if output != want {
		t.Errorf("stdout: got %q, want %q", output, want)
	}
}

func TestHandleSpeakingStatus_Stopped(t *testing.T) {
	l := &MQTTListener{name: "testactor"}

	payload, _ := json.Marshal(commands.Speaking{Who: "testactor", Status: commands.StatusStopped})
	msg := &mockMessage{payload: payload, topic: "speaking/testactor"}

	output := captureStdout(func() {
		l.handleSpeakingStatus(nil, msg)
	})

	want := fmt.Sprintln("stopped speaking")
	if output != want {
		t.Errorf("stdout: got %q, want %q", output, want)
	}
}

func TestHandleSpeakingStatus_InvalidPayload(t *testing.T) {
	l := &MQTTListener{name: "testactor"}

	msg := &mockMessage{payload: []byte("not valid json"), topic: "speaking/testactor"}

	output := captureStdout(func() {
		l.handleSpeakingStatus(nil, msg)
	})

	// No status output expected for invalid JSON.
	if output != "" {
		t.Errorf("expected no stdout output for invalid JSON, got %q", output)
	}
}

func TestHandleSpeakingStatus_UnknownStatus(t *testing.T) {
	l := &MQTTListener{name: "testactor"}

	payload, _ := json.Marshal(commands.Speaking{Who: "testactor", Status: "unknown"})
	msg := &mockMessage{payload: payload, topic: "speaking/testactor"}

	output := captureStdout(func() {
		l.handleSpeakingStatus(nil, msg)
	})

	// No stdout output expected for an unrecognised status.
	if output != "" {
		t.Errorf("expected no stdout output for unknown status, got %q", output)
	}
}
