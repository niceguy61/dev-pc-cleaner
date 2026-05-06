package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CleanResult struct {
	Item         Item
	Status       string
	Error        string
	DeletedBytes int64
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

		deletedBytes, err := removePath(item.Path, item.System)
		if err != nil {
			if deletedBytes > 0 {
				results = append(results, CleanResult{Item: item, Status: "partial", Error: err.Error(), DeletedBytes: deletedBytes})
				continue
			}
			results = append(results, CleanResult{Item: item, Status: "error", Error: err.Error()})
			continue
		}
		results = append(results, CleanResult{Item: item, Status: "deleted", DeletedBytes: deletedBytes})
	}

	return results
}

func removePath(path string, system bool) (int64, error) {
	if path == "" {
		return 0, fmt.Errorf("empty path")
	}
	if !isSafePath(path, system) {
		return 0, fmt.Errorf("protected path")
	}

	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}

	if info.IsDir() {
		return removeDirContents(path)
	}

	if err := os.Remove(path); err != nil {
		// Some tools create read-only files
		if chmodErr := os.Chmod(path, 0o600); chmodErr == nil {
			if err := os.Remove(path); err != nil {
				return 0, err
			}
			return info.Size(), nil
		}
		return 0, err
	}

	// Ensure parent empty dirs are not deleted; keep behavior safe.
	_ = filepath.Clean(path)
	return info.Size(), nil
}

func removeDirContents(path string) (int64, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	var deletedBytes int64
	var removeErrors []error
	for _, entry := range entries {
		child := filepath.Join(path, entry.Name())
		bytes, err := removeAny(child)
		deletedBytes += bytes
		if err != nil {
			removeErrors = append(removeErrors, err)
		}
	}

	return deletedBytes, errors.Join(removeErrors...)
}

func removeAny(path string) (int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}

	if info.IsDir() {
		deletedBytes, err := removeDirContents(path)
		if err != nil {
			return deletedBytes, err
		}
		if err := os.Remove(path); err != nil {
			return deletedBytes, err
		}
		return deletedBytes, nil
	}

	if err := os.Remove(path); err != nil {
		if chmodErr := os.Chmod(path, 0o600); chmodErr == nil {
			if err := os.Remove(path); err != nil {
				return 0, err
			}
			return info.Size(), nil
		}
		return 0, err
	}
	return info.Size(), nil
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
