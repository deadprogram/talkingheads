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

// TestCallTools_NormalizedName verifies that a tool registered as "tool_movement"
// is found when the model outputs the name without underscores ("toolmovement"),
// which is the observed behaviour of Gemma 4 models.
func TestCallTools_NormalizedName_Found(t *testing.T) {
	mock := &mockTool{}
	a := &Actor{
		tools: map[string]Tool{
			"tool_movement": mock,
		},
	}

	toolCalls := []message.ToolCall{
		{
			Type: "function",
			Function: message.ToolFunction{
				Name:      "toolmovement",
				Arguments: map[string]string{"command": "speak"},
			},
		},
	}

	resps := a.callTools(context.Background(), toolCalls)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response via normalized lookup, got %d", len(resps))
	}
	if !mock.called {
		t.Error("expected mock tool to be called")
	}
}

// TestNormalizeToolName covers the normalizeToolName helper.
func TestNormalizeToolName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"tool_movement", "toolmovement"},
		{"toolmovement", "toolmovement"},
		{"Tool_Movement", "toolmovement"},
		{"tool-movement", "toolmovement"},
		{"noop", "noop"},
	}
	for _, c := range cases {
		got := normalizeToolName(c.input)
		if got != c.want {
			t.Errorf("normalizeToolName(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// TestStripActorMarkup_MissingSpaceAfterPeriod verifies that a single period
// glued to a following capitalised word is split with a space, while decimals,
// ellipses and abbreviations are left alone.
func TestStripActorMarkup_MissingSpaceAfterPeriod(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"I like things.I really do", "I like things. I really do"},
		{"Hello world.Then goodbye", "Hello world. Then goodbye"},
		{"one.Two.Three", "one. Two. Three"},
		// Untouched: decimals, ellipses, abbreviations, lowercase follow.
		{"pi is 3.14", "pi is 3.14"},
		{"wait...okay", "wait...okay"},
		{"e.g. this", "e.g. this"},
		{"U.S.A. today", "U.S.A. today"},
		{"already. spaced", "already. spaced"},
		{"end of sentence.", "end of sentence."},
		// JSON-wrapped responses are unwrapped and still get the fix.
		{`{"response": "hello.World"}`, "hello. World"},
	}
	for _, c := range cases {
		got := stripActorMarkup(c.input)
		if got != c.want {
			t.Errorf("stripActorMarkup(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// TestTruncateToSentences verifies that the helper caps the number of
// sentences in its input and drops any partial trailing sentence.
func TestTruncateToSentences(t *testing.T) {
	cases := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"unlimited zero", "One. Two. Three.", 0, "One. Two. Three."},
		{"unlimited negative", "One. Two. Three.", -1, "One. Two. Three."},
		{"cap to one", "One. Two. Three.", 1, "One."},
		{"cap to two", "One. Two. Three.", 2, "One. Two."},
		{"cap above count", "One. Two.", 5, "One. Two."},
		{"mixed terminators", "Hi! How are you? I am fine.", 2, "Hi! How are you?"},
		{"drop partial trailing", "Done. And then", 1, "Done."},
		{"no terminator", "incomplete sentence", 3, "incomplete sentence"},
	}
	for _, c := range cases {
		got := truncateToSentences(c.input, c.max)
		if got != c.want {
			t.Errorf("%s: truncateToSentences(%q, %d) = %q, want %q", c.name, c.input, c.max, got, c.want)
		}
	}
}
