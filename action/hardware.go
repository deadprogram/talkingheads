//go:build tinygo

package main

import (
	"machine"
	"time"
)

func main() {
	uart := machine.Serial
	uart.Configure(machine.UARTConfig{TX: machine.UART_TX_PIN, RX: machine.UART_RX_PIN})
	head = NewHeadLED()

	go lights()
	go action()

	for {
		if uart.Buffered() > 0 {
			data, _ := uart.ReadByte()

			switch data {
			case 13:
				// return key
				cmd := string(input)
				input = input[:0]
				if err := handleCommand(cmd); err != nil {
					uart.Write([]byte("error: " + err.Error() + "\r\n"))
				}

			default:
				// just capture the character
				input = append(input, data)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func lights() {
	for {
		switch mode {
		case "speak":
			head.Alternate(green, blue)
		case "wait":
			head.Green()
		case "headshake":
			head.Red()
		default:
			head.Off()
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func action() {
	svo, _ = NewServo()

	var waitCounter, speakCounter int

	for {
		switch mode {
		case "look":
			svo.SetAngle(targetAngle)
			angle = targetAngle
			mode = "stop"

		case "slowlook":
			angle = movement(angle, targetAngle)
			svo.SetAngle(angle)
			if angle == targetAngle {
				mode = "stop"
			}

		case "wait":
			// Move a small amount once every 5 seconds (25 × 200ms iterations).
			waitCounter++
			if waitCounter >= 25 {
				waitCounter = 0
				jitter := randomInt(-5, 6)
				svo.SetAngle(angle + jitter)
			}

		case "speak":
			// Move a small amount once every second (5 × 200ms iterations).
			speakCounter++
			if speakCounter >= 5 {
				speakCounter = 0
				jitter := randomInt(-10, 11)
				svo.SetAngle(angle + jitter)
			}

		case "headshake":
			// Move back and forth 3 times to indicate "No".
			for i := 0; i < 3; i++ {
				svo.SetAngle(60)
				time.Sleep(300 * time.Millisecond)
				svo.SetAngle(120)
				time.Sleep(300 * time.Millisecond)
			}
			svo.SetAngle(90)
			angle = 90
			mode = "stop"

		case "stop":
			svo.SetAngle(90)
			angle = 90
		}

		time.Sleep(200 * time.Millisecond)
	}
}
