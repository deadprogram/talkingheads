package actor

import (
	"context"
	"testing"

	"github.com/ardanlabs/kronk/sdk/kronk/model"
)

// mockTool is a simple Tool implementation for testing.
type mockTool struct {
	called bool
}

func (m *mockTool) Call(_ context.Context, toolCall model.ResponseToolCall) model.D {
	m.called = true
	return toolSuccessResponse(toolCall.ID, toolCall.Function.Name, "result", "ok")
}

func TestGetMore_NilFunc_ReturnsFalse(t *testing.T) {
	a := &Actor{moreConversationFunc: nil}

	conv := []model.D{}
	got := a.GetMore(&conv)
	if got {
		t.Error("expected GetMore to return false when moreConversationFunc is nil")
	}
}

func TestGetMore_WithFunc_ReturnsTrue(t *testing.T) {
	called := false
	a := &Actor{
		moreConversationFunc: func(conversation *[]model.D) {
			called = true
		},
	}

	conv := []model.D{}
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

	toolCalls := []model.ResponseToolCall{
		{
			ID: "tc-unknown",
			Function: model.ResponseToolCallFunction{
				Name:      "nonexistent_tool",
				Arguments: model.ToolCallArguments{},
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

	toolCalls := []model.ResponseToolCall{
		{
			ID: "tc-1",
			Function: model.ResponseToolCallFunction{
				Name:      "mock_tool",
				Arguments: model.ToolCallArguments{},
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
	if resps[0]["role"] != "tool" {
		t.Errorf("role: got %v, want %q", resps[0]["role"], "tool")
	}
}

func TestCallTools_MixedTools(t *testing.T) {
	mock := &mockTool{}
	a := &Actor{
		tools: map[string]Tool{
			"known_tool": mock,
		},
	}

	toolCalls := []model.ResponseToolCall{
		{ID: "tc-1", Function: model.ResponseToolCallFunction{Name: "known_tool", Arguments: model.ToolCallArguments{}}},
		{ID: "tc-2", Function: model.ResponseToolCallFunction{Name: "unknown_tool", Arguments: model.ToolCallArguments{}}},
	}

	resps := a.callTools(context.Background(), toolCalls)
	if len(resps) != 1 {
		t.Errorf("expected 1 response (unknown skipped), got %d", len(resps))
	}
}
