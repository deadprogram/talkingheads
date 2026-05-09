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

const (
	MatrixIdle = iota
	MatrixPulse
	MatrixVUMeter
)

// Matrix represents the Arduino UNO Q LED matrix display.
type Matrix struct {
	unoqmatrix.Device
	mode int
	stop chan struct{}
}

// NewMatrix initializes the Arduino UNO Q LED matrix display.
func NewMatrix() *Matrix {
	display := unoqmatrix.NewFromBasePin(machine.PF0)
	display.ClearDisplay()

	return &Matrix{Device: display, stop: make(chan struct{})}
}

// StartPulse begins the EKG pulse animation on the Arduino UNO Q LED matrix display.
func (m *Matrix) StartPulse() {
	if m.mode == MatrixPulse {
		return
	}
	if m.mode != MatrixIdle {
		m.stop <- struct{}{}
		time.Sleep(25 * time.Millisecond)
	}
	m.mode = MatrixPulse
	m.ClearDisplay()
	go m.drawPulse()
}

// StartVUMeter begins the VU meter animation on the Arduino UNO Q LED matrix display.
func (m *Matrix) StartVUMeter() {
	if m.mode == MatrixVUMeter {
		return
	}
	if m.mode != MatrixIdle {
		m.stop <- struct{}{}
		time.Sleep(25 * time.Millisecond)
	}
	m.mode = MatrixVUMeter
	m.ClearDisplay()
	go m.drawVUMeter()
}

// Stop halts the animation and clears the Arduino UNO Q LED matrix display.
func (m *Matrix) Stop() {
	if m.mode == MatrixIdle {
		return
	}
	m.mode = MatrixIdle
	m.stop <- struct{}{}
	time.Sleep(25 * time.Millisecond) // Give the draw goroutine time to exit before we clear the display.
	m.ClearDisplay()
}

// draw runs the EKG pulse animation on the Arduino UNO Q LED matrix display.
func (m *Matrix) drawPulse() {
	w, h := m.Size()
	mid := int16(h / 2)

	// EKG heartbeat waveform: baseline, P wave, QRS complex, T wave.
	// jitter is 0 at baseline positions; non-zero values are shifted per-cycle.
	base := []int16{
		mid, mid, mid, mid,
		mid - 1, mid - 1, mid,
		mid + 1, 1, h - 1, mid,
		mid - 1, mid - 1, mid,
		mid, mid, mid, mid, mid, mid,
	}
	isBaseline := []bool{
		true, true, true, true,
		false, false, false,
		false, false, false, false,
		false, false, false,
		true, true, true, true, true, true,
	}
	pattern := make([]int16, len(base))
	copy(pattern, base)

	// cols holds the lit y-position for each display column.
	cols := make([]int16, w)
	for i := range cols {
		cols[i] = mid
	}

	const refreshes = 64 // redraws per frame for brightness
	const frameDelay = 20 * time.Millisecond

	step := 0
	for {
		// At the start of each new cycle, compute a fresh jitter offset.
		if step%len(base) == 0 {
			jitter := int16(rand.Int31n(3)) - 1 // -1, 0, or +1
			for i, v := range base {
				if isBaseline[i] {
					pattern[i] = v
				} else {
					j := v + jitter
					if j < 0 {
						j = 0
					} else if j >= h {
						j = h - 1
					}
					pattern[i] = j
				}
			}
		}

		copy(cols, cols[1:])
		cols[w-1] = pattern[step%len(pattern)]
		step++

		for i := 0; i < refreshes; i++ {
			m.ClearDisplay()
			for x, y := range cols {
				m.SetPixel(int16(x), y, on)
			}
			m.Display()
		}

		select {
		case <-m.stop:
			return
		default:
			time.Sleep(frameDelay)
		}
	}
}

// drawVUMeter runs a VU meter animation on the Arduino UNO Q LED matrix display.
// Each column is an independent bar whose level drifts toward a random target.
func (m *Matrix) drawVUMeter() {
	w, h := m.Size()

	maxLevel := h - 1 // leave top row free for headroom

	levels := make([]int16, w)
	targets := make([]int16, w)
	for i := range levels {
		levels[i] = 0
		targets[i] = int16(rand.Int31n(int32(maxLevel) + 1))
	}

	const refreshes = 64
	const frameDelay = 10 * time.Millisecond
	const targetChangeEvery = 2 // frames between new random targets

	frame := 0
	for {
		if frame%targetChangeEvery == 0 {
			for i := range targets {
				targets[i] = int16(rand.Int31n(int32(maxLevel) + 1))
			}
		}
		frame++

		for i := range levels {
			if levels[i] < targets[i] {
				levels[i]++
			} else if levels[i] > targets[i] {
				levels[i]--
			}
		}

		for r := 0; r < refreshes; r++ {
			m.ClearDisplay()
			for x, lvl := range levels {
				for y := h - 1; y >= h-lvl; y-- {
					m.SetPixel(int16(x), y, on)
				}
			}
			m.Display()
		}

		select {
		case <-m.stop:
			return
		default:
			time.Sleep(frameDelay)
		}
	}
}
