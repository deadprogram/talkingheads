package main

import "image/color"

var PersonalityColor = "blue"

var (
	red    = color.RGBA{R: 0xff, G: 0x00, B: 0x00}
	green  = color.RGBA{R: 0x00, G: 0xff, B: 0x00}
	blue   = color.RGBA{R: 0x00, G: 0x00, B: 0xff}
	purple = color.RGBA{R: 160, G: 32, B: 240}
	orange = color.RGBA{R: 255, G: 165, B: 0}
	yellow = color.RGBA{R: 0xff, G: 0xff, B: 0x00}
	black  = color.RGBA{R: 0x00, G: 0x00, B: 0x00}
)

func personalityColor() color.RGBA {
	switch PersonalityColor {
	case "red":
		return red
	case "green":
		return green
	case "blue":
		return blue
	case "purple":
		return purple
	case "orange":
		return orange
	case "yellow":
		return yellow
	default:
		return red
	}
}

func personalityColorAlternate() color.RGBA {
	switch PersonalityColor {
	case "red":
		return orange
	case "green":
		return yellow
	case "blue":
		return purple
	case "purple":
		return blue
	case "orange":
		return red
	case "yellow":
		return green
	default:
		return orange
	}
}
