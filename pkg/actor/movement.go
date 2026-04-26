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
			"description": "Move the actor to a new position.",
			"parameters": model.D{
				"type": "object",
				"properties": model.D{
					"movement": model.D{
						"type":        "string",
						"description": "What kind of head movement action to perform. Valid values are, 'look', 'shake', 'slowlook'.",
					},
					"direction": model.D{
						"type":        "string",
						"description": "The direction to move. Valid values are, 'left', 'right', 'center'.",
					},
				},
				"required": []string{"movement", "direction"},
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

	movement, _ := toolCall.Function.Arguments["movement"].(string)
	direction, _ := toolCall.Function.Arguments["direction"].(string)
	if movement == "" || direction == "" {
		return toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("missing required parameters"))
	}

	// Here you would implement the logic to move the actor.
	// For now, we'll just simulate it by printing to the console.
	fmt.Printf("Actor performs %s movement towards %s\n", movement, direction)

	return toolSuccessResponse(toolCall.ID, toolCall.Function.Name, "movement", movement, "direction", direction)
}
