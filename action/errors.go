package main

import (
	"errors"
	"time"
)

var (
	errAngleRequired  = errors.New("angle required")
	errInvalidAngle   = errors.New("invalid angle")
	errUnknownCommand = errors.New("unknown command")
)

func failure(msg string) {
	for {
		println(msg)
		time.Sleep(1 * time.Second)
	}
}
