package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CleanResult struct {
	Item   Item
	Status string
	Error  string
}

func Clean(items []Item, apply bool, allowSystemDelete bool) []CleanResult {
	results := make([]CleanResult, 0, len(items))

	for _, item := range items {
		if item.Kind != ItemPath {
			results = append(results, CleanResult{Item: item, Status: "skipped", Error: "unsupported"})
			continue
		}
		if item.System && !allowSystemDelete {
			results = append(results, CleanResult{Item: item, Status: "skipped", Error: "system-protected"})
			continue
		}
		if item.Status == "missing" {
			results = append(results, CleanResult{Item: item, Status: "missing"})
			continue
		}

		if !apply {
			results = append(results, CleanResult{Item: item, Status: "dry-run"})
			continue
		}

		err := removePath(item.Path, item.System)
		if err != nil {
			results = append(results, CleanResult{Item: item, Status: "error", Error: err.Error()})
			continue
		}
		results = append(results, CleanResult{Item: item, Status: "deleted"})
	}

	return results
}

func removePath(path string, system bool) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	if !isSafePath(path, system) {
		return fmt.Errorf("protected path")
	}

	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}

	if err := os.Remove(path); err != nil {
		// Some tools create read-only files
		if chmodErr := os.Chmod(path, 0o600); chmodErr == nil {
			return os.Remove(path)
		}
		return err
	}

	// Ensure parent empty dirs are not deleted; keep behavior safe.
	_ = filepath.Clean(path)
	return nil
}

func isSafePath(path string, system bool) bool {
	p := filepath.Clean(path)
	pLower := strings.ToLower(p)

	vol := strings.ToLower(filepath.VolumeName(p))
	root := vol + "\\"
	if pLower == strings.ToLower(root) {
		return false
	}

	protectedRoots := []string{
		`c:\windows`,
		`c:\program files`,
		`c:\program files (x86)`,
		`c:\programdata`,
	}

	for _, root := range protectedRoots {
		if strings.HasPrefix(pLower, root) {
			if system && isAllowedSystemPath(pLower) {
				return true
			}
			return false
		}
	}

	return true
}

func isAllowedSystemPath(pLower string) bool {
	allowed := []string{
		strings.ToLower(os.ExpandEnv(`%WINDIR%\Temp`)),
		strings.ToLower(os.ExpandEnv(`%WINDIR%\Prefetch`)),
		strings.ToLower(os.ExpandEnv(`%WINDIR%\SoftwareDistribution\Download`)),
		strings.ToLower(os.ExpandEnv(`%PROGRAMDATA%\Microsoft\Windows\DeliveryOptimization\Cache`)),
		strings.ToLower(os.ExpandEnv(`%WINDIR%\Logs`)),
	}
	for _, a := range allowed {
		if a == "" {
			continue
		}
		a = filepath.Clean(a)
		if strings.HasPrefix(pLower, strings.ToLower(a)) {
			return true
		}
	}
	return false
}
