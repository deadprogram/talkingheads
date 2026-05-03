package actor

import (
	"encoding/json"
	"testing"

	"github.com/deadprogram/talkingheads/pkg/commands"
)

// mockCommander records the last command sent.
type mockCommander struct {
	sentCmds []string
}

func (m *mockCommander) Send(cmd string) error {
	m.sentCmds = append(m.sentCmds, cmd)
	return nil
}

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

func TestHandleSpeakingStatus_Speaking(t *testing.T) {
	mc := &mockCommander{}
	l := &MQTTListener{name: "testactor", commander: mc}

	payload, _ := json.Marshal(commands.Speaking{Who: "testactor", Status: commands.StatusSpeaking})
	msg := &mockMessage{payload: payload, topic: "speaking/testactor"}

	l.handleSpeakingStatus(nil, msg)

	if len(mc.sentCmds) != 1 || mc.sentCmds[0] != "speak" {
		t.Errorf("commander.Send: got %v, want [\"speak\"]", mc.sentCmds)
	}
}

func TestHandleSpeakingStatus_Stopped(t *testing.T) {
	mc := &mockCommander{}
	l := &MQTTListener{name: "testactor", commander: mc}

	payload, _ := json.Marshal(commands.Speaking{Who: "testactor", Status: commands.StatusStopped})
	msg := &mockMessage{payload: payload, topic: "speaking/testactor"}

	l.handleSpeakingStatus(nil, msg)

	if len(mc.sentCmds) != 1 || mc.sentCmds[0] != "stop" {
		t.Errorf("commander.Send: got %v, want [\"stop\"]", mc.sentCmds)
	}
}

func TestHandleSpeakingStatus_InvalidPayload(t *testing.T) {
	mc := &mockCommander{}
	l := &MQTTListener{name: "testactor", commander: mc}

	msg := &mockMessage{payload: []byte("not valid json"), topic: "speaking/testactor"}

	l.handleSpeakingStatus(nil, msg)

	// No command expected for invalid JSON.
	if len(mc.sentCmds) != 0 {
		t.Errorf("expected no commands for invalid JSON, got %v", mc.sentCmds)
	}
}

func TestHandleSpeakingStatus_UnknownStatus(t *testing.T) {
	mc := &mockCommander{}
	l := &MQTTListener{name: "testactor", commander: mc}

	payload, _ := json.Marshal(commands.Speaking{Who: "testactor", Status: "unknown"})
	msg := &mockMessage{payload: payload, topic: "speaking/testactor"}

	l.handleSpeakingStatus(nil, msg)

	// No command expected for an unrecognised status.
	if len(mc.sentCmds) != 0 {
		t.Errorf("expected no commands for unknown status, got %v", mc.sentCmds)
	}
}
