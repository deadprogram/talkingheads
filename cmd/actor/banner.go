package main

import (
	"strings"

	bannerpkg "github.com/deadprogram/talkingheads/pkg/banner"
)

// makeBanner renders the startup banner using the provided actor name.
func makeBanner(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "ACTOR"
	}
	return bannerpkg.Generate(strings.ToUpper(name))
}
