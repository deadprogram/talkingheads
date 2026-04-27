package actor

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/ardanlabs/kronk/sdk/kronk/model"
	"go.bug.st/serial"
)

// Commander is the interface used by Movement to send action commands.
// Two implementations are provided: LogCommander (console) and SerialCommander.
type Commander interface {
	Send(cmd string) error
}

// LogCommander writes commands to the standard logger.
type LogCommander struct{}

func (l *LogCommander) Send(cmd string) error {
	log.Printf("action: %s", cmd)
	return nil
}

// SerialCommander sends commands to a microcontroller over a serial port.
type SerialCommander struct {
	port io.ReadWriteCloser
}

// NewSerialCommander opens the named serial port at the given baud rate.
func NewSerialCommander(portName string, baudRate int) (*SerialCommander, error) {
	port, err := serial.Open(portName, &serial.Mode{BaudRate: baudRate})
	if err != nil {
		return nil, fmt.Errorf("opening serial port %s: %w", portName, err)
	}
	return &SerialCommander{port: port}, nil
}

// Send writes the command followed by a carriage return to the serial port.
func (s *SerialCommander) Send(cmd string) error {
	_, err := fmt.Fprintf(s.port, "%s\r", cmd)
	return err
}

// Close closes the underlying serial port.
func (s *SerialCommander) Close() error {
	return s.port.Close()
}

type Movement struct {
	name      string
	commander Commander
}

// RegisterMovement creates a new instance of the Movement tool and loads it
// into the provided tools map. If commander is nil, a LogCommander is used.
func RegisterMovement(tools map[string]Tool, commander Commander) model.D {
	if commander == nil {
		commander = &LogCommander{}
	}
	rf := Movement{
		name:      "tool_movement",
		commander: commander,
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

	var serialCmd string
	var successArgs []any

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
		serialCmd = fmt.Sprintf("%s %d", command, angleInt)
		successArgs = []any{"command", command, "angle", angleInt}
	case "headshake", "wait", "speak", "stop":
		serialCmd = command
		successArgs = []any{"command", command}
	default:
		return toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("unknown command: %s", command))
	}

	if err := rf.commander.Send(serialCmd); err != nil {
		return toolErrorResponse(toolCall.ID, toolCall.Function.Name, fmt.Errorf("sending command: %w", err))
	}
	return toolSuccessResponse(toolCall.ID, toolCall.Function.Name, successArgs...)
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
