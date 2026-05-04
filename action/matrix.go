//go:build arduino_uno_q

package main

import (
	"machine"

	"image/color"
	"math/rand"
	"time"

	"tinygo.org/x/drivers/unoqmatrix"
)

var on = color.RGBA{255, 255, 255, 255}

// Matrix represents the Arduino UNO Q LED matrix display.
type Matrix struct {
	unoqmatrix.Device
	running bool
	stop    chan struct{}
}

// NewMatrix initializes the Arduino UNO Q LED matrix display.
func NewMatrix() *Matrix {
	display := unoqmatrix.NewFromBasePin(machine.PF0)
	display.ClearDisplay()

	return &Matrix{Device: display, stop: make(chan struct{})}
}

// Start begins the animation on the Arduino UNO Q LED matrix display.
func (m *Matrix) Start() {
	if m.running {
		return
	}
	m.running = true
	m.ClearDisplay()
	go m.draw()
}

// Stop halts the animation and clears the Arduino UNO Q LED matrix display.
func (m *Matrix) Stop() {
	if !m.running {
		return
	}
	m.running = false
	m.stop <- struct{}{}
	time.Sleep(5 * time.Millisecond) // Give the draw goroutine time to exit before we clear the display.
	m.ClearDisplay()
}

// draw runs the animation loop for the Arduino UNO Q LED matrix display.
func (m *Matrix) draw() {
	w, h := m.Size()
	x := int16(0)
	y := int16(0)
	deltaX := int16(1)
	deltaY := int16(1)

	for {
		pixel := m.GetPixel(x, y)
		if pixel.R != 0 || pixel.G != 0 || pixel.B != 0 {
			m.ClearDisplay()
			x = 1 + int16(rand.Int31n(3))
			y = 1 + int16(rand.Int31n(3))
			deltaX = 1
			deltaY = 1
			if rand.Int31n(2) == 0 {
				deltaX = -1
			}
			if rand.Int31n(2) == 0 {
				deltaY = -1
			}
		}
		m.SetPixel(x, y, on)

		x += deltaX
		y += deltaY

		if x == 0 || x == w-1 {
			deltaX = -deltaX
		}

		if y == 0 || y == h-1 {
			deltaY = -deltaY
		}

		m.Display()
		select {
		case <-m.stop:
			return
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}
}
