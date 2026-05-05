package main

import (
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

var (
	input  = make([]byte, 0, 64)
	mode   = StateStopped
	modeMu sync.RWMutex

	head        *HeadLED
	matrix      *Matrix
	svo         ServoDevice
	angle       = 90
	targetAngle = 90
)

const (
	// Command constants for serial input.
	CommandLook      = "look"
	CommandSlowLook  = "slowlook"
	CommandWait      = "wait"
	CommandSpeak     = "speak"
	CommandHeadShake = "headshake"
	CommandStop      = "stop"

	// State constants for controlling the behavior of the main loop.
	StateLooking     = "looking"
	StateSlowLooking = "slowlooking"
	StateWaiting     = "waiting"
	StateSpeaking    = "speaking"
	StateHeadShaking = "headshaking"
	StateStopped     = "stopped"
)

// processCommand parses and dispatches a received serial command.
func processCommand(cmd string) error {
	parts := strings.SplitN(strings.TrimSpace(cmd), " ", 2)
	command := parts[0]

	switch command {
	case CommandLook, CommandSlowLook:
		if len(parts) != 2 {
			return errAngleRequired
		}
		a, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return errInvalidAngle
		}
		targetAngle = a
		setMode(stateForCommand(command))
	case CommandWait, CommandSpeak, CommandHeadShake, CommandStop:
		setMode(stateForCommand(command))
	default:
		setMode(StateStopped)
		return errUnknownCommand
	}
	return nil
}

func getMode() string {
	modeMu.RLock()
	defer modeMu.RUnlock()
	return mode
}

func setMode(m string) {
	modeMu.Lock()
	defer modeMu.Unlock()
	mode = m
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

func stateForCommand(command string) string {
	switch command {
	case CommandLook:
		return StateLooking
	case CommandSlowLook:
		return StateSlowLooking
	case CommandWait:
		return StateWaiting
	case CommandSpeak:
		return StateSpeaking
	case CommandHeadShake:
		return StateHeadShaking
	case CommandStop:
		return StateStopped
	default:
		return "unknown"
	}
}
