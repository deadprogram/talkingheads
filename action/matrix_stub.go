//go:build !arduino_uno_q

package main

// Matrix is a stub for non-hardware builds.
type Matrix struct{}

func NewMatrix() *Matrix { return &Matrix{} }

func (m *Matrix) Start() {}
func (m *Matrix) Stop()  {}
