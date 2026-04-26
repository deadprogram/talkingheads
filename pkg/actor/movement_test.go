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
}

func TestMovementCall_ValidArgs(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-1",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"movement": "look", "direction": "left"},
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
		t.Errorf("expected movement in content, got: %s", content)
	}
}

func TestMovementCall_MissingArgs(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-2",
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

func TestMovementCall_MissingDirection(t *testing.T) {
	tools := make(map[string]Tool)
	RegisterMovement(tools)

	toolCall := model.ResponseToolCall{
		ID: "tc-3",
		Function: model.ResponseToolCallFunction{
			Name:      "tool_movement",
			Arguments: model.ToolCallArguments{"movement": "shake"},
		},
	}

	resp := tools["tool_movement"].Call(context.Background(), toolCall)

	content := resp["content"].(string)
	if !strings.Contains(content, "FAILED") {
		t.Errorf("expected FAILED in content, got: %s", content)
	}
}
