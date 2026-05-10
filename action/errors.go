package main

import (
	"errors"
	"time"
)

var (
	errAngleRequired  = errors.New("angle required")
	errInvalidAngle   = errors.New("invalid angle")
	errUnknownCommand = errors.New("unknown command")
	errColorRequired  = errors.New("color required")
	errInvalidColor   = errors.New("invalid color")
)

func failure(msg string) {
	for {
		println(msg)
		time.Sleep(1 * time.Second)
	}
}
