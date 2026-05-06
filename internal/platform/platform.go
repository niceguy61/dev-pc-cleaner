package platform

import (
	"os"
	"runtime"
	"strings"
)

type Kind string

const (
	WindowsNative Kind = "windows-native"
	WSL           Kind = "wsl"
	MacOS         Kind = "macos"
	Linux         Kind = "linux"
	Other         Kind = "other"
)

type Environment struct {
	GOOS string
	Kind Kind
}

func Detect() Environment {
	return DetectFrom(runtime.GOOS, envLookup, readProcVersion())
}

func DetectFrom(goos string, getenv func(string) string, procVersion string) Environment {
	switch goos {
	case "windows":
		return Environment{GOOS: goos, Kind: WindowsNative}
	case "darwin":
		return Environment{GOOS: goos, Kind: MacOS}
	case "linux":
		if isWSL(getenv, procVersion) {
			return Environment{GOOS: goos, Kind: WSL}
		}
		return Environment{GOOS: goos, Kind: Linux}
	default:
		return Environment{GOOS: goos, Kind: Other}
	}
}

func (e Environment) SupportsWindowsWslShrink() bool {
	return e.Kind == WindowsNative
}

func isWSL(getenv func(string) string, procVersion string) bool {
	if strings.TrimSpace(getenv("WSL_DISTRO_NAME")) != "" || strings.TrimSpace(getenv("WSL_INTEROP")) != "" {
		return true
	}
	version := strings.ToLower(procVersion)
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

func envLookup(key string) string {
	return os.Getenv(key)
}

func readProcVersion() string {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	return string(b)
}
