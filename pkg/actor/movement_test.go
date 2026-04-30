package actor

import (
	"context"
	"strings"
	"testing"

	"github.com/hybridgroup/yzma/pkg/message"
)

func TestRegisterMovement_AddsToMap(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	if _, ok := tools["tool_movement"]; !ok {
		t.Fatal("expected tool_movement to be registered in tools map")
	}
}

func TestRegisterMovement_DocumentStructure(t *testing.T) {
	tools := make(map[string]Tool)
	doc := RegisterMovement(tools, nil)

	if doc.Type != "function" {
		t.Errorf("type: got %v, want %q", doc.Type, "function")
	}
	if doc.Function.Name != "tool_movement" {
		t.Errorf("function name: got %v, want %q", doc.Function.Name, "tool_movement")
	}
	if doc.Function.Description == "" {
		t.Error("function description should not be empty")
	}

	params := doc.Function.Parameters
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties field is not map[string]interface{}")
	}
	if _, ok := props["command"]; !ok {
		t.Error("expected 'command' property in tool document")
	}
	if _, ok := props["angle"]; !ok {
		t.Error("expected 'angle' property in tool document")
	}
}

func makeToolCall(name string, args map[string]string) message.ToolCall {
	return message.ToolCall{
		Type: "function",
		Function: message.ToolFunction{
			Name:      name,
			Arguments: args,
		},
	}
}

func TestMovementCall_Look(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	resp := tools["tool_movement"].Call(context.Background(),
		makeToolCall("tool_movement", map[string]string{"command": "look", "angle": "135"}))

	if !strings.Contains(resp, "SUCCESS") {
		t.Errorf("expected SUCCESS in response, got: %s", resp)
	}
	if !strings.Contains(resp, "look") {
		t.Errorf("expected command in response, got: %s", resp)
	}
}

func TestMovementCall_SlowLook(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	resp := tools["tool_movement"].Call(context.Background(),
		makeToolCall("tool_movement", map[string]string{"command": "slowlook", "angle": "45"}))

	if !strings.Contains(resp, "SUCCESS") {
		t.Errorf("expected SUCCESS in response, got: %s", resp)
	}
}

func TestMovementCall_HeadShake(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	resp := tools["tool_movement"].Call(context.Background(),
		makeToolCall("tool_movement", map[string]string{"command": "headshake"}))

	if !strings.Contains(resp, "SUCCESS") {
		t.Errorf("expected SUCCESS in response, got: %s", resp)
	}
}

func TestMovementCall_MissingCommand(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	resp := tools["tool_movement"].Call(context.Background(),
		makeToolCall("tool_movement", map[string]string{}))

	if !strings.Contains(resp, "FAILED") {
		t.Errorf("expected FAILED in response, got: %s", resp)
	}
}

func TestMovementCall_LookMissingAngle(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	resp := tools["tool_movement"].Call(context.Background(),
		makeToolCall("tool_movement", map[string]string{"command": "look"}))

	if !strings.Contains(resp, "FAILED") {
		t.Errorf("expected FAILED in response, got: %s", resp)
	}
}

func TestMovementCall_LookAngleOutOfRange(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	resp := tools["tool_movement"].Call(context.Background(),
		makeToolCall("tool_movement", map[string]string{"command": "look", "angle": "200"}))

	if !strings.Contains(resp, "FAILED") {
		t.Errorf("expected FAILED in response, got: %s", resp)
	}
}

func TestMovementCall_UnknownCommand(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools, nil)

	resp := tools["tool_movement"].Call(context.Background(),
		makeToolCall("tool_movement", map[string]string{"command": "dance"}))

	if !strings.Contains(resp, "FAILED") {
		t.Errorf("expected FAILED in response, got: %s", resp)
	}
}
