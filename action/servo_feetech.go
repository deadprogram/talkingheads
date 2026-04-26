//go:build feetech

package main

import (
	"context"
	"machine"

	"github.com/hipsterbrown/feetech-servo/feetech"
	"github.com/hipsterbrown/feetech-servo/transports"
)

type ServoFeetech struct {
	*feetech.Servo
}

func NewServo() (*ServoFeetech, error) {
	uart := machine.UART1
	uart.Configure(machine.UARTConfig{
		BaudRate: 1000000,
		TX:       machine.UART_TX_PIN,
		RX:       machine.UART_RX_PIN,
	})

	// Create a new servo transport
	transport, err := transports.OpenSerial(transports.SerialConfig{
		Device:   uart,
		BaudRate: 1000000,
	})
	if err != nil {
		failure("Failed to open serial transport:" + err.Error())
	}
	// Create a new servo bus
	bus, err := feetech.NewBus(feetech.BusConfig{
		Transport: transport,
		Protocol:  feetech.ProtocolSTS,
	})
	if err != nil {
		failure("Failed to create bus:" + err.Error())
	}

	servo := feetech.NewServo(bus, 1, nil)

	return &ServoFeetech{
		Servo: servo,
	}, nil
}

func (s *ServoFeetech) SetAngle(angle int) {
	ctx := context.Background()
	s.Enable(ctx)
	s.SetPosition(ctx, calcAngle(angle))
}

func calcAngle(angle int) int {
	if angle < 0 {
		angle = 0
	} else if angle > 180 {
		angle = 180
	}

	// keep to the middle 512 values of the midpoint.
	return 1792 + angle*512/180
}
