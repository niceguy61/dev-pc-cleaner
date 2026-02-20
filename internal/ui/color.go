package ui

import (
	"os"
	"runtime"
)

var colorEnabled = true

func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

func SupportsColor() bool {
	if runtime.GOOS != "windows" {
		return os.Getenv("TERM") != ""
	}
	// Windows Terminal, VSCode, or similar should set these.
	if os.Getenv("WT_SESSION") != "" || os.Getenv("TERM") != "" {
		return true
	}
	return false
}

func Bold(s string) string {
	return wrap("1", s)
}

func Green(s string) string {
	return wrap("32", s)
}

func Red(s string) string {
	return wrap("31", s)
}

func Yellow(s string) string {
	return wrap("33", s)
}

func wrap(code string, s string) string {
	if !colorEnabled {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}
