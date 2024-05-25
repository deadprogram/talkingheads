package main

import (
	"machine"
	"math/rand"

	"time"

	"github.com/hybridgroup/gopherbot"
	"tinygo.org/x/drivers/servo"
)

var (
	uart  = machine.Serial
	tx    = machine.UART_TX_PIN
	rx    = machine.UART_RX_PIN
	input = make([]byte, 0, 64)
	mode  = "off"

	backpack = gopherbot.Backpack()
	head     = NewHeadLED()
	svo      servo.Servo
	position string
)

func main() {
	uart.Configure(machine.UARTConfig{TX: tx, RX: rx})

	go lights()
	go motion()

	for {
		if uart.Buffered() > 0 {
			data, _ := uart.ReadByte()

			switch data {
			case 13:
				// return key
				cmd := string(input)
				input = input[:0]
				switch {
				case cmd == "talk1" || cmd == "talk2" || cmd == "talk3":
					if cmd[4] == position[0] {
						// only talk if we are in the right position
						mode = cmd
						continue
					}
					// everyone else quiet
					mode = "stop"
				default:
					mode = cmd
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
		case "green":
			backpack.Green()
			head.Green()
		case "red":
			backpack.Red()
			head.Red()
		case "talk":
			backpack.Alternate(green, black)
			head.Alternate(green, blue)
		case "talk1":
			backpack.Alternate(green, black)
			head.Alternate(green, black)
		case "talk2":
			backpack.Alternate(blue, black)
			head.Alternate(blue, red)
		case "talk3":
			backpack.Alternate(red, black)
			head.Alternate(red, black)
		case "stop":
			backpack.Off()
			head.Off()
		default:
			backpack.Off()
			head.Off()
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func motion() {
	svo, _ = servo.New(machine.TCC0, machine.A1)

	for {
		switch mode {
		case "talk":
			svo.SetAngle(randomInt(50, 130))
		case "talk1":
			svo.SetAngle(90)
		case "talk2":
			svo.SetAngle(90)
		case "talk3":
			svo.SetAngle(90)
		case "stop":
			svo.SetAngle(90)
		default:
			svo.SetAngle(90)
		}

		time.Sleep(1500 * time.Millisecond)
	}
}

// Returns an int >= min, < max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}
