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

		time.Sleep(200 * time.Millisecond)
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
				svo.SetAngle(75)
				position = "left"
			}
		case "right":
			if position != "right" {
				svo.SetAngle(115)
				position = "right"
			}
		case "stop":
			if position != "center" {
				svo.SetAngle(90)
				position = "center"
			}
		default:
			svo.SetAngle(randomInt(50, 130))
			position = "random"
		}

		time.Sleep(1500 * time.Millisecond)
	}
}

// Returns an int >= min, < max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}
