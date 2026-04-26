//go:build !feetech

package main

import (
	"machine"

	"tinygo.org/x/drivers/servo"
)

var (
	servoPWM = &machine.TIM3
	servoPin = machine.D3
)

type ServoStandard struct {
	servo.Servo
}

func NewServo() (*ServoStandard, error) {
	svo, err := servo.New(servoPWM, servoPin)
	if err != nil {
		return nil, err
	}

	return &ServoStandard{
		Servo: svo,
	}, nil
}

func (s *ServoStandard) SetAngle(angle int) {
	s.Servo.SetAngle(angle)
}
