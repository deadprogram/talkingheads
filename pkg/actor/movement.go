package actor

import (
	"context"
	"fmt"

	"github.com/ardanlabs/kronk/sdk/kronk/model"
)

type Movement struct {
	name string
}

// RegisterMovement creates a new instance of the Movement tool and loads it
// into the provided tools map.
func RegisterMovement(tools map[string]Tool) model.D {
	rf := Movement{
		name: "tool_movement",
	}
	tools[rf.name] = &rf

	return rf.toolDocument()
}

func (rf *Movement) toolDocument() model.D {
	return model.D{
		"type": "function",
		"function": model.D{
			"name":        rf.name,
			"description": "Control the head movement of the actor.",
			"parameters": model.D{
				"type": "object",
				"properties": model.D{
					"command": model.D{
						"type":        "string",
						"description": "The movement command to perform. Valid values are: 'look' (turn to angle), 'slowlook' (slowly turn to angle), 'headshake' (shake head to indicate no), 'wait' (idle movement), 'speak' (movement while speaking), 'stop' (stop and center).",
					},
					"angle": model.D{
						"type":        "integer",
						"description": "The angle in degrees (0-180) to look at. Required for 'look' and 'slowlook' commands. 90 is center, 0 is full right, 180 is full left.",
						"minimum":     0,
						"maximum":     180,
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// Call is the function that is called by the agent to move the actor when the model requests the tool with the specified parameters.
func (rf *Movement) Call(ctx context.Context, toolCall model.ResponseToolCall) (resp model.D) {
	defer func() {
		if r := recover(); r != nil {
			resp = toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("%s", r))
		}
	}()

	command, _ := toolCall.Function.Arguments["command"].(string)
	if command == "" {
		return toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("missing required parameter: command"))
	}

	switch command {
	case "look", "slowlook":
		angle, ok := toolCall.Function.Arguments["angle"]
		if !ok {
			return toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("angle required for %s command", command))
		}
		angleInt, ok := toInt(angle)
		if !ok || angleInt < 0 || angleInt > 180 {
			return toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("angle must be an integer between 0 and 180"))
		}
		fmt.Printf("%s %d\n", command, angleInt)
		return toolSuccessResponse(toolCall.ID, toolCall.Function.Name, "command", command, "angle", angleInt)
	case "headshake", "wait", "speak", "stop":
		fmt.Printf("%s\n", command)
		return toolSuccessResponse(toolCall.ID, toolCall.Function.Name, "command", command)
	default:
		return toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("unknown command: %s", command))
	}
}

// toInt converts a JSON-unmarshalled number (float64) or int to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	}
	return 0, false
}
