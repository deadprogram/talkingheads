package actor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// ---- toolSuccessResponse ----

func TestToolSuccessResponse_Structure(t *testing.T) {
	resp := toolSuccessResponse("id-1", "my_tool", "key", "value")

	if resp["role"] != "tool" {
		t.Errorf("role: got %q, want %q", resp["role"], "tool")
	}
	if resp["name"] != "my_tool" {
		t.Errorf("name: got %q, want %q", resp["name"], "my_tool")
	}
	if resp["tool_call_id"] != "id-1" {
		t.Errorf("tool_call_id: got %q, want %q", resp["tool_call_id"], "id-1")
	}

	content, ok := resp["content"].(string)
	if !ok {
		t.Fatal("content is not a string")
	}

	var payload struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if payload.Status != "SUCCESS" {
		t.Errorf("status: got %q, want %q", payload.Status, "SUCCESS")
	}
	if payload.Data["key"] != "value" {
		t.Errorf("data[key]: got %v, want %q", payload.Data["key"], "value")
	}
}

func TestToolSuccessResponse_MultipleKeyValues(t *testing.T) {
	resp := toolSuccessResponse("id-2", "multi_tool", "a", "1", "b", "2")

	content := resp["content"].(string)
	var payload struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if payload.Data["a"] != "1" || payload.Data["b"] != "2" {
		t.Errorf("unexpected data: %v", payload.Data)
	}
}

// ---- toolErrorResponse ----

func TestToolErrorResponse_Structure(t *testing.T) {
	resp := toolErrorResponse("id-3", "fail_tool", context.DeadlineExceeded)

	if resp["role"] != "tool" {
		t.Errorf("role: got %q, want %q", resp["role"], "tool")
	}
	if resp["name"] != "fail_tool" {
		t.Errorf("name: got %q, want %q", resp["name"], "fail_tool")
	}

	content := resp["content"].(string)
	var payload struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if payload.Status != "FAILED" {
		t.Errorf("status: got %q, want %q", payload.Status, "FAILED")
	}
	if !strings.Contains(payload.Data["error"].(string), "deadline exceeded") {
		t.Errorf("error message not found in data: %v", payload.Data)
	}
}
