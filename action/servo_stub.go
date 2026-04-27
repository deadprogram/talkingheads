//go:build !tinygo

package main

// ServoStub is a no-op ServoDevice for non-hardware builds.
type ServoStub struct{}

func NewServo() (*ServoStub, error) { return &ServoStub{}, nil }

func (s *ServoStub) SetAngle(_ int) {}
