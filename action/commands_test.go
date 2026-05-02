package main

import "testing"

// resetState resets the global command state before each test.
func resetState() {
	mode = StateStopped
	targetAngle = 90
}

func TestProcessCommand_Look(t *testing.T) {
	resetState()
	if err := processCommand("look 135"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != StateLooking {
		t.Errorf("mode = %q, want %q", mode, StateLooking)
	}
	if targetAngle != 135 {
		t.Errorf("targetAngle = %d, want 135", targetAngle)
	}
}

func TestProcessCommand_LookMissingAngle(t *testing.T) {
	resetState()
	if err := processCommand("look"); err != errAngleRequired {
		t.Errorf("err = %v, want errAngleRequired", err)
	}
}

func TestProcessCommand_LookInvalidAngle(t *testing.T) {
	resetState()
	if err := processCommand("look abc"); err != errInvalidAngle {
		t.Errorf("err = %v, want errInvalidAngle", err)
	}
}

func TestProcessCommand_SlowLook(t *testing.T) {
	resetState()
	if err := processCommand("slowlook 45"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != StateSlowLooking {
		t.Errorf("mode = %q, want %q", mode, StateSlowLooking)
	}
	if targetAngle != 45 {
		t.Errorf("targetAngle = %d, want 45", targetAngle)
	}
}

func TestProcessCommand_SlowLookMissingAngle(t *testing.T) {
	resetState()
	if err := processCommand("slowlook"); err != errAngleRequired {
		t.Errorf("err = %v, want errAngleRequired", err)
	}
}

func TestProcessCommand_Wait(t *testing.T) {
	resetState()
	if err := processCommand("wait"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != StateWaiting {
		t.Errorf("mode = %q, want %q", mode, StateWaiting)
	}
}

func TestProcessCommand_Speak(t *testing.T) {
	resetState()
	if err := processCommand("speak"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != StateSpeaking {
		t.Errorf("mode = %q, want %q", mode, StateSpeaking)
	}
}

func TestProcessCommand_Headshake(t *testing.T) {
	resetState()
	if err := processCommand("headshake"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != StateHeadShaking {
		t.Errorf("mode = %q, want %q", mode, StateHeadShaking)
	}
}

func TestProcessCommand_Stop(t *testing.T) {
	resetState()
	if err := processCommand("stop"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != StateStopped {
		t.Errorf("mode = %q, want %q", mode, StateStopped)
	}
}

func TestProcessCommand_UnknownCommand(t *testing.T) {
	resetState()
	if err := processCommand("dance"); err != errUnknownCommand {
		t.Errorf("err = %v, want errUnknownCommand", err)
	}
}

func TestProcessCommand_TrimsWhitespace(t *testing.T) {
	resetState()
	if err := processCommand("  look   90  "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if targetAngle != 90 {
		t.Errorf("targetAngle = %d, want 90", targetAngle)
	}
}

func TestMovement_TowardTargetLargeGap(t *testing.T) {
	result := movement(90, 130)
	if result != 90+maxMovement {
		t.Errorf("movement(90, 130) = %d, want %d", result, 90+maxMovement)
	}
}

func TestMovement_TowardTargetSmallGap(t *testing.T) {
	result := movement(90, 95)
	if result != 95 {
		t.Errorf("movement(90, 95) = %d, want 95", result)
	}
}

func TestMovement_AwayFromTargetLargeGap(t *testing.T) {
	result := movement(90, 50)
	if result != 90-maxMovement {
		t.Errorf("movement(90, 50) = %d, want %d", result, 90-maxMovement)
	}
}

func TestMovement_AwayFromTargetSmallGap(t *testing.T) {
	result := movement(90, 85)
	if result != 85 {
		t.Errorf("movement(90, 85) = %d, want 85", result)
	}
}

func TestMovement_AtTarget(t *testing.T) {
	result := movement(90, 90)
	if result != 90 {
		t.Errorf("movement(90, 90) = %d, want 90", result)
	}
}
