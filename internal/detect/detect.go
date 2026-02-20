package detect

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"windows_cleaner/internal/registry"
)

type Result struct {
	Language  string
	Category  string
	Installed bool
	Version   string
	Command   string
}

func Language(ctx context.Context, lang registry.Language, timeout time.Duration) Result {
	res := Result{
		Language: lang.Name,
		Category: lang.Category,
		Version:  "",
		Command:  "",
	}

	for _, cmd := range lang.Commands {
		commandStr := strings.TrimSpace(cmd.Exe + " " + strings.Join(cmd.Args, " "))
		out, installed, version := tryCommand(ctx, cmd, timeout)
		if installed {
			res.Installed = true
			res.Version = version
			res.Command = commandStr
			if res.Version == "" {
				res.Version = "(version unknown)"
			}
			return res
		}

		// If the command executed but failed to return usable output, still record it.
		if out != "" && res.Command == "" {
			res.Command = commandStr
		}
	}

	if installed, version, cmd := envHint(lang.Name); installed {
		res.Installed = true
		res.Version = version
		res.Command = cmd
		return res
	}

	if res.Command == "" && len(lang.Commands) > 0 {
		first := lang.Commands[0]
		res.Command = strings.TrimSpace(first.Exe + " " + strings.Join(first.Args, " "))
	}

	res.Version = "-"
	return res
}

func tryCommand(ctx context.Context, c registry.Command, timeout time.Duration) (string, bool, string) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, c.Exe, c.Args...)
	outBytes, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(outBytes))

	if err == nil {
		return out, true, firstLine(out)
	}

	if errors.Is(err, exec.ErrNotFound) {
		if ok, path := whereInstalled(cctx, c.Exe); ok {
			return path, true, "(path)"
		}
		return out, false, ""
	}

	// If the process ran but returned a non-zero exit code, still treat as installed.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if out != "" {
			return out, true, firstLine(out)
		}
		return out, true, ""
	}

	if cctx.Err() == context.DeadlineExceeded {
		return out, false, ""
	}

	if out != "" {
		return out, true, firstLine(out)
	}

	return out, false, ""
}

func CommandInstalled(ctx context.Context, c registry.Command, timeout time.Duration) bool {
	_, installed, _ := tryCommand(ctx, c, timeout)
	return installed
}

func firstLine(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "\n")
	line := strings.TrimSpace(parts[0])
	if len(line) > 120 {
		return line[:120] + "..."
	}
	return line
}

func envHint(lang string) (bool, string, string) {
	lang = strings.ToLower(lang)
	envs := []string{}
	switch lang {
	case "python":
		envs = []string{"PYTHONHOME", "CONDA_PREFIX"}
	case "java", "kotlin", "scala", "groovy":
		envs = []string{"JAVA_HOME", "JDK_HOME"}
	case "go":
		envs = []string{"GOROOT", "GOPATH"}
	case "rust":
		envs = []string{"RUSTUP_HOME", "CARGO_HOME"}
	case "c#", "f#", "visual basic .net":
		envs = []string{"DOTNET_ROOT"}
	case "javascript", "typescript":
		envs = []string{"NODE_HOME", "NVM_HOME", "NVM_SYMLINK"}
	case "ruby":
		envs = []string{"RUBY_HOME"}
	case "php":
		envs = []string{"PHP_HOME"}
	case "r":
		envs = []string{"R_HOME"}
	case "julia":
		envs = []string{"JULIA_HOME"}
	case "android":
		envs = []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"}
	case "dart":
		envs = []string{"FLUTTER_HOME", "DART_HOME"}
	}

	for _, key := range envs {
		val := strings.TrimSpace(os.Getenv(key))
		if val == "" {
			continue
		}
		if exists(val) {
			return true, "(env)", key + "=" + val
		}
	}
	return false, "", ""
}

func exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(filepath.Clean(path))
	return err == nil
}

func whereInstalled(ctx context.Context, exe string) (bool, string) {
	cmd := exec.CommandContext(ctx, "where", exe)
	outBytes, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(outBytes))
	if err != nil || out == "" {
		return false, ""
	}
	line := strings.Split(out, "\n")[0]
	line = strings.TrimSpace(line)
	if line == "" {
		return false, ""
	}
	return true, line
}
