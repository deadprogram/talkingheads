package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers/ws2812"
)

const LEDCount = 22

var (
	red    = color.RGBA{R: 0xff, G: 0x00, B: 0x00}
	green  = color.RGBA{R: 0x00, G: 0xff, B: 0x00}
	blue   = color.RGBA{R: 0x00, G: 0x00, B: 0xff}
	purple = color.RGBA{R: 160, G: 32, B: 240}
	black  = color.RGBA{R: 0x00, G: 0x00, B: 0x00}
)

// HeadLED controls the Neopixel LEDs.
type HeadLED struct {
	ws2812.Device
	LED     []color.RGBA
	alt     bool
	forward bool
	pos     int
}

// NewHeadLED returns a new HeadLED.
func NewHeadLED() *HeadLED {
	neo := machine.A3
	neo.Configure(machine.PinConfig{Mode: machine.PinOutput})
	v := ws2812.New(neo)

	return &HeadLED{
		Device: v,
		LED:    make([]color.RGBA, LEDCount),
	}
}

// Show sets the visor to display the current LED array state.
func (v *HeadLED) Show() {
	v.WriteColors(v.LED)
}

// Off turns off all the LEDs.
func (v *HeadLED) Off() {
	v.Clear()
}

// SetColor sets the Visor LEDs to a single color.
func (v *HeadLED) SetColor(color color.RGBA) {
	for i := range v.LED {
		v.LED[i] = color
	}

	v.Show()
}

// Clear clears the visor.
func (v *HeadLED) Clear() {
	v.SetColor(color.RGBA{R: 0x00, G: 0x00, B: 0x00})
}

// Red turns all of the Visor LEDs red.
func (v *HeadLED) Red() {
	v.SetColor(color.RGBA{R: 0xff, G: 0x00, B: 0x00})
}

// Green turns all of the Visor LEDs green.
func (v *HeadLED) Green() {
	v.SetColor(color.RGBA{R: 0x00, G: 0xff, B: 0x00})
}

// Blue turns all of the Visor LEDs blue.
func (v *HeadLED) Blue() {
	v.SetColor(color.RGBA{R: 0x00, G: 0x00, B: 0xff})
}

// Alternate the 2 colors.
func (v *HeadLED) Alternate(color1, color2 color.RGBA) {
	v.alt = !v.alt
	for i := range v.LED {
		v.alt = !v.alt
		if v.alt {
			v.LED[i] = color1
		} else {
			v.LED[i] = color2
		}
	}

	v.Show()
}

// Xmas light style
func (v *HeadLED) Xmas() {
	v.Alternate(color.RGBA{R: 0xff, G: 0x00, B: 0x00}, color.RGBA{R: 0x00, G: 0xff, B: 0x00})
}

// Cylon visor mode.
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
			v.LED[i] = color.RGBA{R: 0xff, G: 0x00, B: 0x00}
			v.LED[i+1] = color.RGBA{R: 0xff, G: 0x00, B: 0x00}
		} else {
			v.LED[i] = color.RGBA{R: 0x00, G: 0x00, B: 0x00}
			v.LED[i+1] = color.RGBA{R: 0x00, G: 0x00, B: 0x00}
		}
	}

	v.Show()
}
