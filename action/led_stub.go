//go:build !tinygo

package main

// HeadLED is a stub for non-hardware builds.
type HeadLED struct{}

func NewHeadLED() *HeadLED { return &HeadLED{} }
