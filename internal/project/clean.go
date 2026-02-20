package project

import (
	"os"
	"strings"
)

func Clean(items []Item, apply bool) []CleanResult {
	results := make([]CleanResult, 0, len(items))
	for _, item := range items {
		if item.Kind != ItemPath {
			results = append(results, CleanResult{Item: item, Status: "skipped", Error: "unsupported"})
			continue
		}
		if !apply {
			results = append(results, CleanResult{Item: item, Status: "dry-run"})
			continue
		}
		if err := removePath(item.Path); err != nil {
			results = append(results, CleanResult{Item: item, Status: "error", Error: err.Error()})
			continue
		}
		results = append(results, CleanResult{Item: item, Status: "deleted"})
	}
	return results
}

type CleanResult struct {
	Item   Item
	Status string
	Error  string
}

func removePath(path string) error {
	if path == "" {
		return os.ErrInvalid
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.RemoveAll(path)
	}
	if err := os.Remove(path); err != nil {
		if chmodErr := os.Chmod(path, 0o600); chmodErr == nil {
			return os.Remove(path)
		}
		return err
	}
	_ = strings.TrimSpace(path)
	return nil
}
