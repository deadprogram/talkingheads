package actor

import (
	"context"
	"encoding/json"

	"github.com/hybridgroup/yzma/pkg/message"
)

// Tool describes the features which all tools must implement.
type Tool interface {
	Call(ctx context.Context, toolCall message.ToolCall) string
}

// toolSuccessResponse returns a successful structured tool response as a JSON string.
func toolSuccessResponse(keyValues ...any) string {
	data := make(map[string]any, len(keyValues)/2)
	for i := 0; i < len(keyValues); i += 2 {
		data[keyValues[i].(string)] = keyValues[i+1]
	}

	return toolContentJSON(data, "SUCCESS")
}

// toolErrorResponse returns a failed structured tool response as a JSON string.
func toolErrorResponse(err error) string {
	return toolContentJSON(map[string]any{"error": err.Error()}, "FAILED")
}

// toolContentJSON encodes data and status into a JSON string.
func toolContentJSON(data map[string]any, status string) string {
	info := struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}{
		Status: status,
		Data:   data,
	}

	content, err := json.Marshal(info)
	if err != nil {
		return `{"status":"FAILED","data":{"error":"error marshaling tool response"}}`
	}

	return string(content)
}
