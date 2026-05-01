package main

import (
	"math/rand"
	"strconv"
	"strings"
)

var (
	input = make([]byte, 0, 64)
	mode  = "stop"

	head        *HeadLED
	matrix      *Matrix
	svo         ServoDevice
	angle       = 90
	targetAngle = 90
)

// handleCommand parses and dispatches a received serial command.
func handleCommand(cmd string) error {
	parts := strings.SplitN(strings.TrimSpace(cmd), " ", 2)
	command := parts[0]

	switch command {
	case "look", "slowlook":
		if len(parts) != 2 {
			return errAngleRequired
		}
		a, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return errInvalidAngle
		}
		targetAngle = a
		mode = command
	case "wait", "speak", "headshake", "stop":
		mode = command
	default:
		return errUnknownCommand
	}
	return nil
}

// Returns an int >= min, < max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

const maxMovement = 15

// keep movement to maxMovement degrees at a time
func movement(current, target int) int {
	if current < target {
		if target-current > maxMovement {
			return current + maxMovement
		}
		return target
	} else if current > target {
		if current-target > maxMovement {
			return current - maxMovement
		}
		return target
	}
	return current
}
