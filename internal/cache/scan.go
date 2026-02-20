package cache

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ScanOptions struct {
	ShowMissing   bool
	IncludeSystem bool
	Timeout       time.Duration
}

type ScanResult struct {
	Items []Item
}

func Scan(ctx context.Context, rules []Rule, installed map[string]bool, opts ScanOptions) ScanResult {
	items := make([]Item, 0, len(rules))

	for _, r := range rules {
		if r.System && !opts.IncludeSystem {
			continue
		}
		if len(r.Languages) > 0 && !anyInstalled(installed, r.Languages) {
			continue
		}

		for _, raw := range r.Paths {
			paths := expandPattern(raw)
			if len(paths) == 0 {
				if opts.ShowMissing {
					items = append(items, Item{
						Name:     r.Name,
						Category: r.Category,
						System:   r.System,
						Priority: "-",
						Path:     raw,
						Status:   "missing",
						Kind:     ItemPath,
					})
				}
				continue
			}

			for _, p := range paths {
				info, err := os.Stat(p)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						if opts.ShowMissing {
							items = append(items, Item{
								Name:     r.Name,
								Category: r.Category,
								System:   r.System,
								Priority: "-",
								Path:     p,
								Status:   "missing",
								Kind:     ItemPath,
							})
						}
						continue
					}
					items = append(items, Item{
						Name:     r.Name,
						Category: r.Category,
						System:   r.System,
						Priority: "-",
						Path:     p,
						Status:   "error",
						Kind:     ItemPath,
					})
					continue
				}

				size, count, walkErr := dirSize(p, info)
				status := "ok"
				if walkErr != nil {
					status = "partial"
				}

				items = append(items, Item{
					Name:      r.Name,
					Category:  r.Category,
					System:    r.System,
					Priority:  priorityFor(size),
					Path:      p,
					SizeBytes: size,
					FileCount: count,
					Status:    status,
					Kind:      ItemPath,
				})
			}
		}
	}

	return ScanResult{Items: items}
}

func anyInstalled(installed map[string]bool, langs []string) bool {
	for _, l := range langs {
		if installed[strings.ToLower(l)] {
			return true
		}
	}
	return false
}

func dirSize(path string, info os.FileInfo) (int64, int64, error) {
	if !info.IsDir() {
		return info.Size(), 1, nil
	}

	var size int64
	var count int64
	var walkErr error

	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			walkErr = err
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			walkErr = err
			return nil
		}
		size += fi.Size()
		count++
		return nil
	})

	return size, count, walkErr
}

func expandPattern(raw string) []string {
	if strings.Contains(raw, "$Recycle.Bin") {
		return expandRecycleBins(raw)
	}
	expanded := expandPath(raw)
	if expanded == "" {
		return nil
	}
	if strings.ContainsAny(expanded, "*?") {
		matches, err := filepath.Glob(expanded)
		if err != nil {
			return nil
		}
		return matches
	}
	return []string{expanded}
}

func expandRecycleBins(raw string) []string {
	expanded := expandPath(raw)
	paths := []string{}
	if expanded != "" {
		paths = append(paths, expanded)
	}

	seen := map[string]bool{}
	out := make([]string, 0, 26)
	for _, p := range paths {
		seen[strings.ToLower(p)] = true
		out = append(out, p)
	}

	for d := 'C'; d <= 'Z'; d++ {
		path := string(d) + ":\\$Recycle.Bin"
		if seen[strings.ToLower(path)] {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			seen[strings.ToLower(path)] = true
			out = append(out, path)
		}
	}

	return out
}

func expandPath(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "~") {
		home := os.Getenv("USERPROFILE")
		if home != "" {
			s = filepath.Join(home, strings.TrimPrefix(s, "~"))
		}
	}

	s = expandWindowsEnv(s)
	s = os.ExpandEnv(s)
	cleaned := filepath.Clean(s)
	if cleaned == "." && strings.Contains(raw, "%") {
		return ""
	}
	return cleaned
}

func expandWindowsEnv(s string) string {
	for {
		start := strings.Index(s, "%")
		if start == -1 {
			return s
		}
		end := strings.Index(s[start+1:], "%")
		if end == -1 {
			return s
		}
		end = start + 1 + end
		key := s[start+1 : end]
		val := os.Getenv(key)
		if val == "" {
			return s
		}
		s = s[:start] + val + s[end+1:]
	}
}

func priorityFor(size int64) string {
	if size <= 0 {
		return "Low"
	}
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	switch {
	case size >= 1*gb:
		return "High"
	case size >= 200*mb:
		return "Medium"
	default:
		return "Low"
	}
}
