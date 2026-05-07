//go:build tinygo

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers/ws2812"
)

const LEDCount = 22

// HeadLED controls the WS2812 LEDs.
type HeadLED struct {
	ws2812.Device
	LED        []color.RGBA
	cyclePhase uint8
	cycleDir   int
	forward    bool
	pos        int
}

// NewHeadLED returns a new HeadLED.
func NewHeadLED() *HeadLED {
	neo := machine.D4
	neo.Configure(machine.PinConfig{Mode: machine.PinOutput})
	v := ws2812.New(neo)

	return &HeadLED{
		Device: v,
		LED:    make([]color.RGBA, LEDCount),
	}
}

// Show sets the head LEDs to display the current LED array state.
func (v *HeadLED) Show() {
	v.WriteColors(v.LED)
}

// Off turns off all the LEDs.
func (v *HeadLED) Off() {
	v.Clear()
}

// SetColor sets the head LEDs to a single color.
func (v *HeadLED) SetColor(color color.RGBA) {
	for i := range v.LED {
		v.LED[i] = color
	}

	v.Show()
}

// Clear clears the head.
func (v *HeadLED) Clear() {
	v.SetColor(color.RGBA{R: 0x00, G: 0x00, B: 0x00})
}

// Red turns all of the head LEDs red.
func (v *HeadLED) Red() {
	v.SetColor(red)
}

// Green turns all of the head LEDs green.
func (v *HeadLED) Green() {
	v.SetColor(green)
}

// Blue turns all of the head LEDs blue.
func (v *HeadLED) Blue() {
	v.SetColor(blue)
}

// Alternate smoothly cycles all LEDs between color1 and color2.
// Each call advances the blend phase; call at a fixed rate (e.g. 100ms) for smooth transitions.
func (v *HeadLED) Alternate(color1, color2 color.RGBA) {
	const step = 16 // phase increment per call; 100ms * (256/16) * 2 ≈ 3.2s per full cycle
	if v.cycleDir == 0 {
		v.cycleDir = 1
	}
	next := int(v.cyclePhase) + v.cycleDir*step
	if next >= 255 {
		next = 255
		v.cycleDir = -1
	} else if next <= 0 {
		next = 0
		v.cycleDir = 1
	}
	v.cyclePhase = uint8(next)

	t := v.cyclePhase
	t2 := 255 - t
	c := color.RGBA{
		R: uint8((uint16(color1.R)*uint16(t2) + uint16(color2.R)*uint16(t)) / 255),
		G: uint8((uint16(color1.G)*uint16(t2) + uint16(color2.G)*uint16(t)) / 255),
		B: uint8((uint16(color1.B)*uint16(t2) + uint16(color2.B)*uint16(t)) / 255),
	}
	for i := range v.LED {
		v.LED[i] = c
	}
	v.Show()
}

// Xmas light style
func (v *HeadLED) Xmas() {
	v.Alternate(red, green)
}

// Cylon head mode.
func (v *HeadLED) Cylon() {
	if v.forward {
		v.pos += 2
		if v.pos >= LEDCount {
			v.pos = LEDCount - 2
			v.forward = false
		}
	} else {
		v.pos -= 2
		if v.pos < 0 {
			v.pos = 0
			v.forward = true
		}
	}

	for i := 0; i < LEDCount; i += 2 {
		if i == v.pos {
			v.LED[i] = red
			v.LED[i+1] = red
		} else {
			v.LED[i] = black
			v.LED[i+1] = black
		}
	}

	v.Show()
}
