package actor

import (
	"context"
	"testing"

	"github.com/hybridgroup/yzma/pkg/message"
)

// mockTool is a simple Tool implementation for testing.
type mockTool struct {
	called bool
}

func (m *mockTool) Call(_ context.Context, toolCall message.ToolCall) string {
	m.called = true
	return toolSuccessResponse("result", "ok")
}

func TestGetMore_NilFunc_ReturnsFalse(t *testing.T) {
	a := &Actor{moreConversationFunc: nil}

	conv := []message.Message{}
	got := a.GetMore(&conv)
	if got {
		t.Error("expected GetMore to return false when moreConversationFunc is nil")
	}
}

func TestGetMore_WithFunc_ReturnsTrue(t *testing.T) {
	called := false
	a := &Actor{
		moreConversationFunc: func(conversation *[]message.Message) {
			called = true
		},
	}

	conv := []message.Message{}
	got := a.GetMore(&conv)
	if !got {
		t.Error("expected GetMore to return true when moreConversationFunc is set")
	}
	if !called {
		t.Error("expected moreConversationFunc to be called")
	}
}

func TestCallTools_UnknownTool_Skipped(t *testing.T) {
	a := &Actor{tools: make(map[string]Tool)}

	toolCalls := []message.ToolCall{
		{
			Type: "function",
			Function: message.ToolFunction{
				Name:      "nonexistent_tool",
				Arguments: map[string]string{},
			},
		},
	}

	resps := a.callTools(context.Background(), toolCalls)
	if len(resps) != 0 {
		t.Errorf("expected 0 responses for unknown tool, got %d", len(resps))
	}
}

func TestCallTools_KnownTool_Called(t *testing.T) {
	mock := &mockTool{}
	a := &Actor{
		tools: map[string]Tool{
			"mock_tool": mock,
		},
	}

	toolCalls := []message.ToolCall{
		{
			Type: "function",
			Function: message.ToolFunction{
				Name:      "mock_tool",
				Arguments: map[string]string{},
			},
		},
	}

	resps := a.callTools(context.Background(), toolCalls)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	if !mock.called {
		t.Error("expected mock tool to be called")
	}
	tr, ok := resps[0].(message.ToolResponse)
	if !ok {
		t.Fatalf("expected ToolResponse, got %T", resps[0])
	}
	if tr.Role != "tool" {
		t.Errorf("role: got %q, want %q", tr.Role, "tool")
	}
}

func TestCallTools_MixedTools(t *testing.T) {
	mock := &mockTool{}
	a := &Actor{
		tools: map[string]Tool{
			"known_tool": mock,
		},
	}

	toolCalls := []message.ToolCall{
		{Type: "function", Function: message.ToolFunction{Name: "known_tool", Arguments: map[string]string{}}},
		{Type: "function", Function: message.ToolFunction{Name: "unknown_tool", Arguments: map[string]string{}}},
	}

	resps := a.callTools(context.Background(), toolCalls)
	if len(resps) != 1 {
		t.Errorf("expected 1 response (unknown skipped), got %d", len(resps))
	}
}
