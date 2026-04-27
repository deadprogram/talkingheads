package main

import "testing"

// resetState resets the global command state before each test.
func resetState() {
	mode = "stop"
	targetAngle = 90
}

func TestHandleCommand_Look(t *testing.T) {
	resetState()
	if err := handleCommand("look 135"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "look" {
		t.Errorf("mode = %q, want %q", mode, "look")
	}
	if targetAngle != 135 {
		t.Errorf("targetAngle = %d, want 135", targetAngle)
	}
}

func TestHandleCommand_LookMissingAngle(t *testing.T) {
	resetState()
	if err := handleCommand("look"); err != errAngleRequired {
		t.Errorf("err = %v, want errAngleRequired", err)
	}
}

func TestHandleCommand_LookInvalidAngle(t *testing.T) {
	resetState()
	if err := handleCommand("look abc"); err != errInvalidAngle {
		t.Errorf("err = %v, want errInvalidAngle", err)
	}
}

func TestHandleCommand_SlowLook(t *testing.T) {
	resetState()
	if err := handleCommand("slowlook 45"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "slowlook" {
		t.Errorf("mode = %q, want %q", mode, "slowlook")
	}
	if targetAngle != 45 {
		t.Errorf("targetAngle = %d, want 45", targetAngle)
	}
}

func TestHandleCommand_SlowLookMissingAngle(t *testing.T) {
	resetState()
	if err := handleCommand("slowlook"); err != errAngleRequired {
		t.Errorf("err = %v, want errAngleRequired", err)
	}
}

func TestHandleCommand_Wait(t *testing.T) {
	resetState()
	if err := handleCommand("wait"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "wait" {
		t.Errorf("mode = %q, want %q", mode, "wait")
	}
}

func TestHandleCommand_Speak(t *testing.T) {
	resetState()
	if err := handleCommand("speak"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "speak" {
		t.Errorf("mode = %q, want %q", mode, "speak")
	}
}

func TestHandleCommand_Headshake(t *testing.T) {
	resetState()
	if err := handleCommand("headshake"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "headshake" {
		t.Errorf("mode = %q, want %q", mode, "headshake")
	}
}

func TestHandleCommand_Stop(t *testing.T) {
	resetState()
	if err := handleCommand("stop"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "stop" {
		t.Errorf("mode = %q, want %q", mode, "stop")
	}
}

func TestHandleCommand_UnknownCommand(t *testing.T) {
	resetState()
	if err := handleCommand("dance"); err != errUnknownCommand {
		t.Errorf("err = %v, want errUnknownCommand", err)
	}
}

func TestHandleCommand_TrimsWhitespace(t *testing.T) {
	resetState()
	if err := handleCommand("  look   90  "); err != nil {
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
