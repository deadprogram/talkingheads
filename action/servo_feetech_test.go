//go:build feetech

package main

import "testing"

func TestCalcAngle_Zero(t *testing.T) {
	if got := calcAngle(0); got != 1792 {
		t.Errorf("calcAngle(0) = %d, want 1792", got)
	}
}

func TestCalcAngle_180(t *testing.T) {
	want := 1792 + 180*512/180 // 2304
	if got := calcAngle(180); got != want {
		t.Errorf("calcAngle(180) = %d, want %d", got, want)
	}
}

func TestCalcAngle_90(t *testing.T) {
	want := 1792 + 90*512/180 // midpoint: 2048
	if got := calcAngle(90); got != want {
		t.Errorf("calcAngle(90) = %d, want %d", got, want)
	}
}

func TestCalcAngle_ClampBelowZero(t *testing.T) {
	if got := calcAngle(-10); got != 1792 {
		t.Errorf("calcAngle(-10) = %d, want 1792 (clamped to 0)", got)
	}
}

func TestCalcAngle_ClampAbove180(t *testing.T) {
	want := 1792 + 512 // 2304
	if got := calcAngle(200); got != want {
		t.Errorf("calcAngle(200) = %d, want %d (clamped to 180)", got, want)
	}
}
