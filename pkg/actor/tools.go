package actor

import (
	"context"
	"encoding/json"

	"github.com/ardanlabs/kronk/sdk/kronk/model"
)

// Tool describes the features which all tools must implement.
type Tool interface {
	Call(ctx context.Context, toolCall model.ResponseToolCall) model.D
}

// toolSuccessResponse returns a successful structured tool response.
func toolSuccessResponse(toolID string, toolName string, keyValues ...any) model.D {
	data := make(map[string]any, len(keyValues)/2)
	for i := 0; i < len(keyValues); i += 2 {
		data[keyValues[i].(string)] = keyValues[i+1]
	}

	return toolResponse(toolID, toolName, data, "SUCCESS")
}

// toolErrorResponse returns a failed structured tool response.
func toolErrorResponse(toolID string, toolName string, err error) model.D {
	data := map[string]any{"error": err.Error()}

	return toolResponse(toolID, toolName, data, "FAILED")
}

// toolResponse creates a structured tool response.
func toolResponse(toolID string, toolName string, data map[string]any, status string) model.D {
	info := struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}{
		Status: status,
		Data:   data,
	}

	content, err := json.Marshal(info)
	if err != nil {
		return model.D{
			"role":         "tool",
			"name":         toolName,
			"tool_call_id": toolID,
			"content":      `{"status": "FAILED", "data": "error marshaling tool response"}`,
		}
	}

	return model.D{
		"role":         "tool",
		"name":         toolName,
		"tool_call_id": toolID,
		"content":      string(content),
	}
}
