//go:build tinygo && !(feetech || noservo)

package main

import (
	"machine"

	"tinygo.org/x/drivers/servo"
)

type ServoStandard struct {
	servo.Servo
}

func NewServo() (*ServoStandard, error) {
	servoPWM := &machine.TIM3
	servoPin := machine.D3

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
	machine.A1.Low()
}
