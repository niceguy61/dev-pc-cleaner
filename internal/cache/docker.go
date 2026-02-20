package cache

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type DockerStat struct {
	Label       string
	Total       string
	Active      string
	Size        string
	Reclaimable string
}

func ScanDocker(ctx context.Context, timeout time.Duration) []Item {
	stats, ok := dockerSystemDF(ctx, timeout)
	if !ok {
		return nil
	}

	items := make([]Item, 0, len(stats))
	for _, s := range stats {
		items = append(items, Item{
			Name:      "docker " + strings.ToLower(s.Label),
			Category:  "Docker",
			System:    false,
			Priority:  "-",
			Path:      "docker system df",
			SizeBytes: -1,
			SizeText:  s.Size,
			FileCount: -1,
			Status:    "ok",
			Kind:      ItemDocker,
		})
	}

	return items
}

func dockerSystemDF(ctx context.Context, timeout time.Duration) ([]DockerStat, bool) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, err := dockerSystemDFFormat(cctx)
	if err != nil || out == "" {
		cmd := exec.CommandContext(cctx, "docker", "system", "df")
		outBytes, derr := cmd.CombinedOutput()
		out = strings.TrimSpace(string(outBytes))
		if derr != nil {
			if errors.Is(derr, exec.ErrNotFound) {
				return nil, false
			}
			if out == "" {
				return nil, false
			}
		}
	}

	lines := strings.Split(out, "\n")
	var stats []DockerStat
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "TYPE") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			parts := strings.Split(line, "\t")
			if len(parts) >= 5 {
				stats = append(stats, DockerStat{
					Label:       strings.TrimSpace(parts[0]),
					Total:       strings.TrimSpace(parts[1]),
					Active:      strings.TrimSpace(parts[2]),
					Size:        strings.TrimSpace(parts[3]),
					Reclaimable: strings.TrimSpace(parts[4]),
				})
			}
			continue
		}
		stats = append(stats, DockerStat{
			Label:       fields[0],
			Total:       fields[1],
			Active:      fields[2],
			Size:        fields[3],
			Reclaimable: strings.Join(fields[4:], " "),
		})
	}

	if len(stats) == 0 {
		return nil, false
	}

	return stats, true
}

func dockerSystemDFFormat(ctx context.Context) (string, error) {
	format := "{{.Type}}\t{{.TotalCount}}\t{{.Active}}\t{{.Size}}\t{{.Reclaimable}}"
	cmd := exec.CommandContext(ctx, "docker", "system", "df", "--format", format)
	outBytes, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(outBytes)), err
}

func DockerReclaimableBytes(ctx context.Context, timeout time.Duration) (int64, bool) {
	stats, ok := dockerSystemDF(ctx, timeout)
	if !ok {
		return 0, false
	}
	var total int64
	for _, s := range stats {
		if b, ok := ParseSizeString(s.Reclaimable); ok {
			total += b
		}
	}
	return total, true
}

func ParseSizeString(s string) (int64, bool) {
	if s == "" {
		return 0, false
	}
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return 0, false
	}
	val := fields[0]
	unit := ""
	for i := len(val) - 1; i >= 0; i-- {
		if val[i] < '0' || val[i] > '9' {
			unit = val[i:]
			val = val[:i]
			break
		}
	}
	val = strings.TrimSpace(val)
	unit = strings.TrimSpace(unit)
	if val == "" {
		return 0, false
	}

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false
	}

	mult := float64(1)
	switch strings.ToUpper(unit) {
	case "B":
		mult = 1
	case "KB":
		mult = 1024
	case "MB":
		mult = 1024 * 1024
	case "GB":
		mult = 1024 * 1024 * 1024
	case "TB":
		mult = 1024 * 1024 * 1024 * 1024
	case "":
		mult = 1
	default:
		return 0, false
	}

	return int64(f * mult), true
}

type DockerCleanOptions struct {
	All     bool
	Volumes bool
}

func CleanDocker(ctx context.Context, timeout time.Duration, apply bool, opts DockerCleanOptions) CleanResult {
	item := Item{
		Name:     "docker system prune",
		Category: "Docker",
		Path:     "docker system prune",
		Kind:     ItemDocker,
	}

	if !apply {
		return CleanResult{Item: item, Status: "dry-run"}
	}

	args := []string{"system", "prune", "-f"}
	if opts.All {
		args = append(args, "--all")
	}
	if opts.Volumes {
		args = append(args, "--volumes")
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "docker", args...)
	outBytes, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(outBytes))

	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return CleanResult{Item: item, Status: "missing", Error: "docker not found"}
		}
		if cctx.Err() == context.DeadlineExceeded {
			return CleanResult{Item: item, Status: "error", Error: "timeout"}
		}
		if out != "" {
			return CleanResult{Item: item, Status: "error", Error: trimOneLine(out)}
		}
		return CleanResult{Item: item, Status: "error", Error: err.Error()}
	}

	if out == "" {
		out = "completed"
	}
	return CleanResult{Item: item, Status: "deleted", Error: trimOneLine(out)}
}

func trimOneLine(s string) string {
	line := strings.Split(s, "\n")[0]
	line = strings.TrimSpace(line)
	if len(line) > 120 {
		return fmt.Sprintf("%s...", line[:120])
	}
	return line
}
