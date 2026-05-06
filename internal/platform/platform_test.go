package platform

import "testing"

func TestDetectFromWindowsNative(t *testing.T) {
	env := DetectFrom("windows", emptyEnv, "")
	if env.Kind != WindowsNative {
		t.Fatalf("kind = %q, want %q", env.Kind, WindowsNative)
	}
	if !env.SupportsWindowsWslShrink() {
		t.Fatal("windows native should support Windows WSL shrink instructions")
	}
}

func TestDetectFromWSLEnv(t *testing.T) {
	env := DetectFrom("linux", mapEnv(map[string]string{"WSL_DISTRO_NAME": "Ubuntu"}), "")
	if env.Kind != WSL {
		t.Fatalf("kind = %q, want %q", env.Kind, WSL)
	}
	if env.SupportsWindowsWslShrink() {
		t.Fatal("WSL should not offer Windows-native shrink execution")
	}
}

func TestDetectFromWSLProcVersion(t *testing.T) {
	env := DetectFrom("linux", emptyEnv, "Linux version 5.15.0-microsoft-standard-WSL2")
	if env.Kind != WSL {
		t.Fatalf("kind = %q, want %q", env.Kind, WSL)
	}
}

func TestDetectFromLinux(t *testing.T) {
	env := DetectFrom("linux", emptyEnv, "Linux version 6.8.0-generic")
	if env.Kind != Linux {
		t.Fatalf("kind = %q, want %q", env.Kind, Linux)
	}
}

func TestDetectFromMacOS(t *testing.T) {
	env := DetectFrom("darwin", emptyEnv, "")
	if env.Kind != MacOS {
		t.Fatalf("kind = %q, want %q", env.Kind, MacOS)
	}
}

func emptyEnv(string) string {
	return ""
}

func mapEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
