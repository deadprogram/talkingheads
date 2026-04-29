package actor

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/hybridgroup/yzma/pkg/message"
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
func RegisterMovement(tools map[string]Tool, commander Commander) message.ToolDefinition {
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

func (rf *Movement) toolDocument() message.ToolDefinition {
	return message.ToolDefinition{
		Type: "function",
		Function: message.ToolFunctionDefinition{
			Name:        rf.name,
			Description: "Control the head movement of the actor.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The movement command to perform. Valid values are: 'look' (turn to angle), 'slowlook' (slowly turn to angle), 'headshake' (shake head to indicate no), 'wait' (idle movement), 'speak' (movement while speaking), 'stop' (stop and center).",
					},
					"angle": map[string]interface{}{
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
func (rf *Movement) Call(ctx context.Context, toolCall message.ToolCall) (resp string) {
	defer func() {
		if r := recover(); r != nil {
			resp = toolErrorResponse(fmt.Errorf("%s", r))
		}
	}()

	command := toolCall.Function.Arguments["command"]
	if command == "" {
		return toolErrorResponse(fmt.Errorf("missing required parameter: command"))
	}

	var serialCmd string
	var successArgs []any

	switch command {
	case "look", "slowlook":
		angleStr, ok := toolCall.Function.Arguments["angle"]
		if !ok {
			return toolErrorResponse(fmt.Errorf("angle required for %s command", command))
		}
		angleInt, err := strconv.Atoi(angleStr)
		if err != nil || angleInt < 0 || angleInt > 180 {
			return toolErrorResponse(fmt.Errorf("angle must be an integer between 0 and 180"))
		}
		serialCmd = fmt.Sprintf("%s %d", command, angleInt)
		successArgs = []any{"command", command, "angle", angleInt}
	case "headshake", "wait", "speak", "stop":
		serialCmd = command
		successArgs = []any{"command", command}
	default:
		return toolErrorResponse(fmt.Errorf("unknown command: %s", command))
	}

	if err := rf.commander.Send(serialCmd); err != nil {
		return toolErrorResponse(fmt.Errorf("sending command: %w", err))
	}
	return toolSuccessResponse(successArgs...)
}
