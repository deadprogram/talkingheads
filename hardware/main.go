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
	angle    int
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
				mode = cmd

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

		time.Sleep(100 * time.Millisecond)
	}
}

func motion() {
	svo, _ = servo.New(machine.TCC0, machine.A1)

	for {
		switch mode {
		case "off":
			break
		case "left":
			if position != "left" {
				svo.SetAngle(randomInt(110, 140))
				position = "left"
			}
		case "right":
			if position != "right" {
				svo.SetAngle(randomInt(40, 70))
				position = "right"
			}
		case "talk", "talk1", "talk2", "talk3":
			angle = movement(angle, randomInt(50, 130))
			svo.SetAngle(angle)
			position = "random"
		case "stop":
			if position != "center" {
				svo.SetAngle(90)
				position = "center"
			}
		default:
			if position != "center" {
				svo.SetAngle(90)
				position = "center"
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
}

// Returns an int >= min, < max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

const maxMovement = 15

// keep movement to maxMovement degrees at time
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
