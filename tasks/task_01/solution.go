package main

import (
	"strings"
)

func greet(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Hello, World!"
	}
	return "Hello, " + trimmed + "!"
}
