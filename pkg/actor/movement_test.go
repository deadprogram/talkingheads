package actor

import (
	"context"
	"strings"
	"testing"

	"github.com/ardanlabs/kronk/sdk/kronk/model"
)

func TestRegisterMovement_AddsToMap(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	if _, ok := tools["tool_movement"]; !ok {
		t.Fatal("expected tool_movement to be registered in tools map")
	}
}

func TestRegisterMovement_DocumentStructure(t *testing.T) {
	tools := make(map[string]Tool)
	doc := RegisterMovement(tools)

	if doc["type"] != "function" {
		t.Errorf("type: got %v, want %q", doc["type"], "function")
	}

	fn, ok := doc["function"].(model.D)
	if !ok {
		t.Fatal("function field is not model.D")
	}
	if fn["name"] != "tool_movement" {
		t.Errorf("function name: got %v, want %q", fn["name"], "tool_movement")
	}
	if fn["description"] == "" {
		t.Error("function description should not be empty")
	}

	params, ok := fn["parameters"].(model.D)
	if !ok {
		t.Fatal("parameters field is not model.D")
	}
	props, ok := params["properties"].(model.D)
	if !ok {
		t.Fatal("properties field is not model.D")
	}
	if _, ok := props["command"]; !ok {
		t.Error("expected 'command' property in tool document")
	}
	if _, ok := props["angle"]; !ok {
		t.Error("expected 'angle' property in tool document")
	}
}

func TestMovementCall_Look(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-1",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "look", "angle": float64(135)},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	if resp["role"] != "tool" {
		t.Errorf("role: got %v, want %q", resp["role"], "tool")
	}
	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "SUCCESS") {
		t.Errorf("expected SUCCESS in content, got: %s", content)
	}
	if !strings.Contains(content, "look") {
		t.Errorf("expected command in content, got: %s", content)
	}
}

func TestMovementCall_SlowLook(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-2",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "slowlook", "angle": float64(45)},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "SUCCESS") {
		t.Errorf("expected SUCCESS in content, got: %s", content)
	}
}

func TestMovementCall_HeadShake(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-3",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "headshake"},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "SUCCESS") {
		t.Errorf("expected SUCCESS in content, got: %s", content)
	}
}

func TestMovementCall_Wait(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-4",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "wait"},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "SUCCESS") {
		t.Errorf("expected SUCCESS in content, got: %s", content)
	}
}

func TestMovementCall_Speak(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-5",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "speak"},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "SUCCESS") {
		t.Errorf("expected SUCCESS in content, got: %s", content)
	}
}

func TestMovementCall_Stop(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-6",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "stop"},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "SUCCESS") {
		t.Errorf("expected SUCCESS in content, got: %s", content)
	}
}

func TestMovementCall_MissingCommand(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-7",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "FAILED") {
		t.Errorf("expected FAILED in content, got: %s", content)
	}
}

func TestMovementCall_LookMissingAngle(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-8",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "look"},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "FAILED") {
		t.Errorf("expected FAILED in content, got: %s", content)
	}
}

func TestMovementCall_LookAngleOutOfRange(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-9",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "look", "angle": float64(200)},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "FAILED") {
		t.Errorf("expected FAILED in content, got: %s", content)
	}
}

func TestMovementCall_UnknownCommand(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-10",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"command": "dance"},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}
	if !strings.Contains(content, "FAILED") {
		t.Errorf("expected FAILED in content, got: %s", content)
	}
}
