package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"windows_cleaner/internal/cache"
	"windows_cleaner/internal/detect"
	"windows_cleaner/internal/platform"
	"windows_cleaner/internal/project"
	"windows_cleaner/internal/registry"
	"windows_cleaner/internal/ui"
)

const defaultTimeout = 1500 * time.Millisecond

const (
	defaultCmdMax  = 60
	defaultPathMax = 80
)

type Summary struct {
	ItemCount int   `json:"itemCount"`
	FileCount int64 `json:"fileCount"`
	SizeBytes int64 `json:"sizeBytes"`
}

type ProjectSummary struct {
	Project   string `json:"project"`
	Items     int    `json:"items"`
	FileCount int64  `json:"files"`
	SizeBytes int64  `json:"sizeBytes"`
}

type Config struct {
	NoColor           *bool   `json:"noColor"`
	Timeout           *string `json:"timeout"`
	ShowMissing       *bool   `json:"showMissing"`
	IncludeSystem     *bool   `json:"includeSystem"`
	Apply             *bool   `json:"apply"`
	DockerPrune       *bool   `json:"dockerPrune"`
	DockerAll         *bool   `json:"dockerAll"`
	DockerVolumes     *bool   `json:"dockerVolumes"`
	AllowSystemDelete *bool   `json:"allowSystemDelete"`
	MinMB             *int64  `json:"minMB"`
	MinFiles          *int64  `json:"minFiles"`
	ProjectRoot       *string `json:"projectRoot"`
	ProjectDepth      *int    `json:"projectDepth"`
	ProjectClean      *bool   `json:"projectClean"`
	ProjectReview     *bool   `json:"projectReview"`
	ProjectExclude    *string `json:"projectExclude"`
	ProjectNoDefault  *bool   `json:"projectNoDefaultExclude"`
	RecycleBinOnly    *bool   `json:"recycleBinOnly"`
	Output            *string `json:"output"`
	CmdMax            *int    `json:"cmdMax"`
	PathMax           *int    `json:"pathMax"`
}

func main() {
	args := os.Args[1:]
	cmd := "scan"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd = args[0]
		args = args[1:]
	}

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	noColor := fs.Bool("no-color", false, "disable ANSI colors")
	timeout := fs.Duration("timeout", defaultTimeout, "per-command timeout")
	showMissing := fs.Bool("show-missing", false, "show missing cache paths")
	includeSystem := fs.Bool("include-system", true, "include system-level cache paths")
	apply := fs.Bool("apply", false, "apply destructive changes (clean only)")
	dockerPrune := fs.Bool("docker-prune", true, "run docker system prune during clean")
	dockerAll := fs.Bool("docker-all", false, "prune all unused images (requires -docker-prune)")
	dockerVolumes := fs.Bool("docker-volumes", false, "prune unused volumes (requires -docker-prune)")
	allowSystemDelete := fs.Bool("allow-system-delete", false, "allow deletion of system cache paths (requires -apply)")
	minMB := fs.Int64("min-mb", 0, "filter items smaller than this size (MB)")
	minFiles := fs.Int64("min-files", 0, "filter items with fewer files than this count")
	projectRoot := fs.String("project-root", "", "scan project caches under this root")
	projectMaxDepth := fs.Int("project-depth", 5, "max directory depth for project scan")
	projectClean := fs.Bool("project-clean", false, "include project cache clean")
	projectReview := fs.Bool("project-review", true, "review and select project items before cleaning")
	projectExclude := fs.String("project-exclude", "", "comma-separated exclude paths for project scan")
	projectNoDefault := fs.Bool("project-no-default-exclude", false, "disable default excludes in project scan")
	recycleBinOnly := fs.Bool("recycle-bin-only", false, "only scan/clean recycle bin (system scope)")
	cmdMax := fs.Int("cmd-max", defaultCmdMax, "max width for command column (0 = no limit)")
	pathMax := fs.Int("path-max", defaultPathMax, "max width for path column (0 = no limit)")
	output := fs.String("output", "table", "output format: table|json|csv")
	configPath := fs.String("config", "", "load config from file")
	saveConfig := fs.String("save-config", "", "save config to file")
	_ = fs.Parse(args)

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if *configPath != "" {
		cfg, err := loadConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(2)
		}
		applyConfig(cfg, setFlags, noColor, timeout, showMissing, includeSystem, apply, dockerPrune, dockerAll, dockerVolumes, allowSystemDelete, minMB, minFiles, projectRoot, projectMaxDepth, projectClean, projectReview, projectExclude, projectNoDefault, recycleBinOnly, output, cmdMax, pathMax)
	}

	if *saveConfig != "" {
		cfg := currentConfig(*noColor, *timeout, *showMissing, *includeSystem, *apply, *dockerPrune, *dockerAll, *dockerVolumes, *allowSystemDelete, *minMB, *minFiles, *projectRoot, *projectMaxDepth, *projectClean, *projectReview, *projectExclude, *projectNoDefault, *recycleBinOnly, *output, *cmdMax, *pathMax)
		if err := saveConfigFile(*saveConfig, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save config: %v\n", err)
			os.Exit(2)
		}
	}

	if *output != "table" {
		*noColor = true
	}
	ui.SetColorEnabled(!*noColor && ui.SupportsColor())

	switch cmd {
	case "scan":
		runScan(*timeout, *showMissing, *includeSystem, *minMB, *minFiles, *output, *projectRoot, *projectMaxDepth, *projectExclude, *projectNoDefault, *recycleBinOnly, *cmdMax, *pathMax)
	case "detect":
		runDetect(*timeout, *output, *cmdMax)
	case "clean":
		runClean(*timeout, *showMissing, *includeSystem, *minMB, *minFiles, *apply, *allowSystemDelete, *dockerPrune, *dockerAll, *dockerVolumes, *output, *projectRoot, *projectMaxDepth, *projectClean, *projectReview, *projectExclude, *projectNoDefault, *recycleBinOnly, *cmdMax, *pathMax)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(2)
	}
}

func runDetect(timeout time.Duration, output string, cmdMax int) {
	ctx := context.Background()

	results := detectLanguages(ctx, timeout)
	printLanguageTable(results, output, cmdMax)
}

func printUsage() {
	ui.Println("windows_cleaner <command> [options]")
	ui.Println("")
	ui.Println("Commands:")
	ui.Println("  scan           Detect languages + scan cache locations (default)")
	ui.Println("  detect         Detect installed languages only")
	ui.Println("  clean          Remove cache locations (dry-run unless -apply)")
	ui.Println("  help           Show this help")
	ui.Println("")
	ui.Println("Options:")
	ui.Println("  -no-color            Disable ANSI colors")
	ui.Println("  -timeout             Per-command timeout (e.g. 2s, 1500ms)")
	ui.Println("  -show-missing        Include missing cache paths in output")
	ui.Println("  -include-system      Include system-level cache paths")
	ui.Println("  -apply               Apply destructive changes (clean only)")
	ui.Println("  -allow-system-delete Allow deletion of system cache paths (requires -apply)")
	ui.Println("  -docker-prune         Run docker system prune during clean")
	ui.Println("  -docker-all           Prune all unused images (requires -docker-prune)")
	ui.Println("  -docker-volumes       Prune unused volumes (requires -docker-prune)")
	ui.Println("  -min-mb              Filter items smaller than this size (MB)")
	ui.Println("  -min-files           Filter items with fewer files than this count")
	ui.Println("  -project-root        Scan project caches under this root")
	ui.Println("  -project-depth       Max directory depth for project scan (-1 = unlimited)")
	ui.Println("  -project-clean       Include project cache clean")
	ui.Println("  -project-review      Review and select project items before cleaning")
	ui.Println("  -project-exclude     Comma-separated exclude paths for project scan")
	ui.Println("  -project-no-default-exclude Disable default excludes in project scan")
	ui.Println("  -recycle-bin-only    Only scan/clean recycle bin (system scope)")
	ui.Println("  -cmd-max             Max width for command column (0 = no limit)")
	ui.Println("  -path-max            Max width for path column (0 = no limit)")
	ui.Println("  -output              Output format: table|json|csv")
	ui.Println("  -config              Load config from file")
	ui.Println("  -save-config         Save config to file")
}

func runScan(timeout time.Duration, showMissing bool, includeSystem bool, minMB int64, minFiles int64, output string, projectRoot string, projectDepth int, projectExclude string, projectNoDefault bool, recycleBinOnly bool, cmdMax int, pathMax int) {
	ctx := context.Background()

	results := detectLanguages(ctx, timeout)
	if output == "json" {
		if recycleBinOnly {
			includeSystem = true
		}
		installed := installedMap(results)
		cacheRes := cache.Scan(ctx, cache.Rules(), installed, cache.ScanOptions{
			ShowMissing:   showMissing,
			IncludeSystem: includeSystem,
			Timeout:       timeout,
		})
		cacheRes.Items = append(cacheRes.Items, cache.ScanDocker(ctx, timeout)...)
		cacheRes.Items = filterItems(cacheRes.Items, minMB, minFiles)
		if recycleBinOnly {
			cacheRes.Items = filterRecycleBin(cacheRes.Items)
		}
		projectItems := []project.Item{}
		if !recycleBinOnly {
			projectItems = scanProjectItems(projectRoot, projectDepth, projectExclude, projectNoDefault)
		}
		printJSON(struct {
			Languages      []detect.Result  `json:"languages"`
			Items          []cache.Item     `json:"items"`
			Summary        Summary          `json:"summary"`
			Project        []project.Item   `json:"project"`
			ProjectSummary []ProjectSummary `json:"projectSummary"`
		}{Languages: results, Items: cacheRes.Items, Summary: summarize(cacheRes.Items), Project: projectItems, ProjectSummary: summarizeProjects(projectItems)})
		return
	}
	printLanguageTable(results, output, cmdMax)

	if recycleBinOnly {
		includeSystem = true
	}
	installed := installedMap(results)
	cacheRes := cache.Scan(ctx, cache.Rules(), installed, cache.ScanOptions{
		ShowMissing:   showMissing,
		IncludeSystem: includeSystem,
		Timeout:       timeout,
	})
	cacheRes.Items = append(cacheRes.Items, cache.ScanDocker(ctx, timeout)...)
	cacheRes.Items = filterItems(cacheRes.Items, minMB, minFiles)
	if recycleBinOnly {
		cacheRes.Items = filterRecycleBin(cacheRes.Items)
	}
	printCacheTable(cacheRes.Items, output, pathMax)

	projectItems := []project.Item{}
	if !recycleBinOnly {
		projectItems = scanProjectItems(projectRoot, projectDepth, projectExclude, projectNoDefault)
	}
	printProjectTable(projectItems, output, pathMax)
	printProjectGroups(projectItems)
}

func runClean(timeout time.Duration, showMissing bool, includeSystem bool, minMB int64, minFiles int64, apply bool, allowSystemDelete bool, dockerPrune bool, dockerAll bool, dockerVolumes bool, output string, projectRoot string, projectDepth int, projectClean bool, projectReview bool, projectExclude string, projectNoDefault bool, recycleBinOnly bool, cmdMax int, pathMax int) {
	ctx := context.Background()
	env := platform.Detect()

	results := detectLanguages(ctx, timeout)
	if apply && output == "table" {
		if confirmSkip("system cache (include-system)") {
			includeSystem = false
			allowSystemDelete = false
		}
		if confirmSkip("docker prune") {
			dockerPrune = false
		}
	}
	if recycleBinOnly {
		includeSystem = true
	}
	installed := installedMap(results)
	cacheRes := cache.Scan(ctx, cache.Rules(), installed, cache.ScanOptions{
		ShowMissing:   showMissing,
		IncludeSystem: includeSystem,
		Timeout:       timeout,
	})
	if dockerPrune {
		cacheRes.Items = append(cacheRes.Items, cache.ScanDocker(ctx, timeout)...)
	}
	cacheRes.Items = filterItems(cacheRes.Items, minMB, minFiles)
	if recycleBinOnly {
		cacheRes.Items = filterRecycleBin(cacheRes.Items)
	}
	projectItems := []project.Item{}
	if !recycleBinOnly {
		projectItems = scanProjectItems(projectRoot, projectDepth, projectExclude, projectNoDefault)
	}
	sizeByPath := buildSizeMap(cacheRes.Items, projectItems)
	var estimatePlan int64
	var estimateSet bool

	if output == "json" {
		cleanResults := cache.Clean(cacheRes.Items, apply, allowSystemDelete && apply)
		projectCleanResults := project.Clean(projectItems, apply && projectClean)
		var dockerRes *cache.CleanResult
		if dockerPrune {
			res := cache.CleanDocker(ctx, timeout, apply, cache.DockerCleanOptions{
				All:     dockerAll,
				Volumes: dockerVolumes,
			})
			dockerRes = &res
		}
		printJSON(struct {
			Languages      []detect.Result       `json:"languages"`
			Items          []cache.Item          `json:"items"`
			Summary        Summary               `json:"summary"`
			Clean          []cache.CleanResult   `json:"clean"`
			DockerPrune    *cache.CleanResult    `json:"dockerPrune,omitempty"`
			Project        []project.Item        `json:"project"`
			ProjectSummary []ProjectSummary      `json:"projectSummary"`
			ProjectClean   []project.CleanResult `json:"projectClean"`
		}{Languages: results, Items: cacheRes.Items, Summary: summarize(cacheRes.Items), Clean: cleanResults, DockerPrune: dockerRes, Project: projectItems, ProjectSummary: summarizeProjects(projectItems), ProjectClean: projectCleanResults})
		return
	}
	printLanguageTable(results, output, cmdMax)

	printCacheTable(cacheRes.Items, output, pathMax)
	printProjectTable(projectItems, output, pathMax)
	printProjectGroups(projectItems)

	projectItemsToClean := projectItems
	var projectCleanResults []project.CleanResult
	if projectClean && apply && output == "table" && projectReview {
		projectItemsToClean = selectProjectItems(projectItems)
	}

	if apply && output == "table" {
		scope := buildScopeString(cacheRes.Items, projectItems, projectClean, includeSystem, allowSystemDelete, dockerPrune)
		estimate := estimateBytes(cacheRes.Items, projectItems, projectClean)
		if dockerPrune {
			if dockerEst, ok := cache.DockerReclaimableBytes(ctx, timeout); ok {
				estimate += dockerEst
			}
		}
		estimatePlan = estimate
		estimateSet = true
		ui.Println(ui.Bold("Plan"))
		ui.Println(fmt.Sprintf("Scan complete. About to clean %s. Scope: %s", humanBytes(estimate), scope))
		ui.Println("Category summary: " + categorySummaryLine(cacheRes.Items, projectItems, projectClean))
		ui.Println("Top items: " + topItemsLine(cacheRes.Items, projectItems, projectClean, 5))
		ui.Println("Proceed? [y/n]")
		if !confirmYesNo() {
			ui.Println(ui.Yellow("Aborted by user"))
			return
		}
	}

	ui.Println(ui.Bold("Clean Results"))
	cleanResults := cache.Clean(cacheRes.Items, apply, allowSystemDelete && apply)
	printCleanTable(cleanResults, output, pathMax)
	var actualCleaned int64
	if apply && output == "table" {
		cleaned := deletedBytes(cleanResults, sizeByPath)
		actualCleaned += cleaned
		ui.Println(fmt.Sprintf("Done: about %s cleaned.", humanBytes(cleaned)))
	}

	if projectClean {
		ui.Println(ui.Bold("Project Clean Results"))
		projectCleanResults = project.Clean(projectItemsToClean, apply)
		printProjectCleanTable(projectCleanResults, output, pathMax)
		if apply && output == "table" {
			cleaned := deletedProjectBytes(projectCleanResults, sizeByPath)
			actualCleaned += cleaned
			ui.Println(fmt.Sprintf("Project clean: about %s cleaned.", humanBytes(cleaned)))
		}
	}

	if dockerPrune {
		ui.Println(ui.Bold("Docker Prune"))
		dockerRes := cache.CleanDocker(ctx, timeout, apply, cache.DockerCleanOptions{
			All:     dockerAll,
			Volumes: dockerVolumes,
		})
		printCleanTable([]cache.CleanResult{dockerRes}, output, pathMax)
		if apply && output == "table" {
			if reclaimed := parseDockerReclaimed(dockerRes); reclaimed > 0 {
				actualCleaned += reclaimed
				ui.Println(fmt.Sprintf("Docker clean: about %s reclaimed.", humanBytes(reclaimed)))
			}
		}
	}

	if apply && output == "table" && estimateSet {
		diff := actualCleaned - estimatePlan
		if diff < 0 {
			diff = -diff
		}
		if shouldWarnDiff(estimatePlan, diff) {
			ui.Println(ui.Yellow(fmt.Sprintf("Warning: estimate vs actual differs by %s.", humanBytes(diff))))
		}
	}

	if apply && output == "table" && actualCleaned > 0 && env.SupportsWindowsWslShrink() {
		ui.Println("")
		ui.Println("WSL disk shrink can reduce VHDX file size after cleanup.")
		ui.Println("Show shrink instructions? [y/n]")
		if confirmYesNo() {
			ui.Println("Run this in Administrator PowerShell:")
			ui.Println("  wsl --shutdown")
			ui.Println("  .\\scripts\\shrink_wsl.ps1 -Force")
		}
	}
}

func estimateBytes(cacheItems []cache.Item, projectItems []project.Item, includeProject bool) int64 {
	var total int64
	for _, it := range cacheItems {
		if it.SizeBytes > 0 {
			total += it.SizeBytes
		}
	}
	if includeProject {
		for _, it := range projectItems {
			if it.SizeBytes > 0 {
				total += it.SizeBytes
			}
		}
	}
	return total
}

func buildScopeString(cacheItems []cache.Item, projectItems []project.Item, includeProject bool, includeSystem bool, allowSystemDelete bool, dockerPrune bool) string {
	parts := []string{"cache"}
	if includeProject && len(projectItems) > 0 {
		parts = append(parts, "project")
	}
	if includeSystem {
		if allowSystemDelete {
			parts = append(parts, "system")
		} else {
			parts = append(parts, "system (scan-only)")
		}
	}
	if hasRecycleBin(cacheItems) {
		parts = append(parts, "recycle bin")
	}
	if dockerPrune {
		parts = append(parts, "docker")
	}
	return strings.Join(parts, ", ")
}

func categorySummaryLine(cacheItems []cache.Item, projectItems []project.Item, includeProject bool) string {
	type agg struct {
		Size int64
	}
	by := map[string]*agg{}
	for _, it := range cacheItems {
		if it.SizeBytes > 0 {
			a := by[it.Category]
			if a == nil {
				a = &agg{}
				by[it.Category] = a
			}
			a.Size += it.SizeBytes
		}
	}
	if includeProject {
		for _, it := range projectItems {
			if it.SizeBytes > 0 {
				a := by["Project"]
				if a == nil {
					a = &agg{}
					by["Project"] = a
				}
				a.Size += it.SizeBytes
			}
		}
	}

	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+":"+humanBytes(by[k].Size))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func topItemsLine(cacheItems []cache.Item, projectItems []project.Item, includeProject bool, n int) string {
	type entry struct {
		Name string
		Size int64
	}
	list := make([]entry, 0, len(cacheItems)+len(projectItems))
	for _, it := range cacheItems {
		if it.SizeBytes > 0 {
			list = append(list, entry{Name: it.Name, Size: it.SizeBytes})
		}
	}
	if includeProject {
		for _, it := range projectItems {
			if it.SizeBytes > 0 {
				list = append(list, entry{Name: it.Name, Size: it.SizeBytes})
			}
		}
	}
	if len(list) == 0 {
		return "-"
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Size > list[j].Size })
	if n <= 0 || n > len(list) {
		n = len(list)
	}
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, list[i].Name+":"+humanBytes(list[i].Size))
	}
	return strings.Join(parts, ", ")
}

func hasRecycleBin(items []cache.Item) bool {
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Path), "\\$recycle.bin") {
			return true
		}
	}
	return false
}

func confirmSkip(label string) bool {
	fmt.Printf("Exclude %s? [y/N]: ", label)
	return confirmYesNo()
}

func confirmYesNo() bool {
	var resp string
	_, _ = fmt.Fscanln(os.Stdin, &resp)
	resp = normalizeYesNo(resp)
	return resp == "y" || resp == "yes"
}

func normalizeYesNo(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "ㅛ":
		return "y"
	case "ㅜ":
		return "n"
	default:
		return s
	}
}

func selectProjectItems(items []project.Item) []project.Item {
	if len(items) == 0 {
		return nil
	}
	ui.Println(ui.Bold("Project Review"))
	for i, it := range items {
		fmt.Printf("%d. %s | %s | %s | %s | %s\n", i+1, it.Name, it.Project, it.Category, humanBytes(it.SizeBytes), it.Path)
	}
	fmt.Print("Select items (e.g. 1,3,5 or 'all'/'none' or 'gt:500mb' or 'cat:web' or 'project:<name>'): ")
	var line string
	_, _ = fmt.Fscanln(os.Stdin, &line)
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" || line == "none" {
		return nil
	}
	if line == "all" {
		return items
	}
	if strings.HasPrefix(line, "gt:") {
		min := strings.TrimSpace(strings.TrimPrefix(line, "gt:"))
		if minBytes, ok := parseSize(min); ok {
			filtered := make([]project.Item, 0, len(items))
			for _, it := range items {
				if it.SizeBytes >= minBytes {
					filtered = append(filtered, it)
				}
			}
			return filtered
		}
	}
	if strings.HasPrefix(line, "cat:") {
		cat := strings.TrimSpace(strings.TrimPrefix(line, "cat:"))
		if cat != "" {
			filtered := make([]project.Item, 0, len(items))
			for _, it := range items {
				if strings.ToLower(it.Category) == cat {
					filtered = append(filtered, it)
				}
			}
			return filtered
		}
	}
	if strings.HasPrefix(line, "project:") {
		proj := strings.TrimSpace(strings.TrimPrefix(line, "project:"))
		if proj != "" {
			filtered := make([]project.Item, 0, len(items))
			for _, it := range items {
				if strings.ToLower(it.Project) == proj {
					filtered = append(filtered, it)
				}
			}
			return filtered
		}
	}
	parts := strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == ' ' })
	seen := map[int]bool{}
	selected := make([]project.Item, 0, len(parts))
	for _, p := range parts {
		idx := 0
		_, err := fmt.Sscanf(p, "%d", &idx)
		if err != nil {
			continue
		}
		idx--
		if idx < 0 || idx >= len(items) {
			continue
		}
		if seen[idx] {
			continue
		}
		seen[idx] = true
		selected = append(selected, items[idx])
	}
	return selected
}

func parseSize(s string) (int64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, false
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "kb"):
		mult = 1024
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "mb"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "gb"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "gb")
	}
	s = strings.TrimSpace(s)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return int64(v * float64(mult)), true
}

func detectLanguages(ctx context.Context, timeout time.Duration) []detect.Result {
	langs := registry.Languages()
	sort.Slice(langs, func(i, j int) bool { return strings.ToLower(langs[i].Name) < strings.ToLower(langs[j].Name) })

	results := make([]detect.Result, 0, len(langs))
	for _, lang := range langs {
		res := detect.Language(ctx, lang, timeout)
		results = append(results, res)
	}
	return results
}

func installedMap(results []detect.Result) map[string]bool {
	out := map[string]bool{}
	for _, r := range results {
		out[strings.ToLower(r.Language)] = r.Installed
	}
	return out
}

func printLanguageTable(results []detect.Result, output string, cmdMax int) {
	headers := []string{"Language", "Category", "Status", "Version", "Command"}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		status := ui.Red("Not Found")
		if r.Installed {
			status = ui.Green("Installed")
		}
		rows = append(rows, []string{r.Language, r.Category, status, r.Version, truncate(r.Command, cmdMax)})
	}

	switch output {
	case "json":
		printJSON(struct {
			Languages []detect.Result `json:"languages"`
		}{Languages: results})
	case "csv":
		printCSV(headers, rows)
	default:
		ui.Println(ui.Bold("Language Detection"))
		fmt.Print(ui.RenderTable(headers, rows))
	}
}

func printCacheTable(items []cache.Item, output string, pathMax int) {
	headers := []string{"Item", "Category", "Priority", "Status", "Size", "Files", "Path"}
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		status := renderCacheStatus(it)
		size := formatSize(it)
		files := formatCount(it.FileCount, it.Kind)
		rows = append(rows, []string{it.Name, it.Category, it.Priority, status, size, files, truncate(it.Path, pathMax)})
	}

	switch output {
	case "json":
		printJSON(struct {
			Items   []cache.Item `json:"items"`
			Summary Summary      `json:"summary"`
		}{Items: items, Summary: summarize(items)})
	case "csv":
		printCSV(headers, rows)
	default:
		ui.Println(ui.Bold("Cache Scan"))
		fmt.Print(ui.RenderTable(headers, rows))
		printSummary(items)
		printCategorySummary(items)
		printTopItems(items)
	}
}

func printProjectTable(items []project.Item, output string, pathMax int) {
	if len(items) == 0 {
		return
	}
	headers := []string{"Item", "Project", "Category", "Status", "Size", "Files", "Path"}
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		status := it.Status
		if status == "" {
			status = "ok"
		}
		rows = append(rows, []string{
			it.Name,
			it.Project,
			it.Category,
			status,
			humanBytes(it.SizeBytes),
			fmt.Sprintf("%d", it.FileCount),
			truncate(it.Path, pathMax),
		})
	}

	switch output {
	case "json":
		printJSON(struct {
			Items []project.Item `json:"items"`
		}{Items: items})
	case "csv":
		printCSV(headers, rows)
	default:
		ui.Println(ui.Bold("Project Caches"))
		fmt.Print(ui.RenderTable(headers, rows))
	}
}

func printProjectGroups(items []project.Item) {
	if len(items) == 0 {
		return
	}
	headers := []string{"Project", "Items", "Files", "Size"}
	rows := make([][]string, 0, len(items))
	for _, s := range summarizeProjects(items) {
		rows = append(rows, []string{
			s.Project,
			fmt.Sprintf("%d", s.Items),
			fmt.Sprintf("%d", s.FileCount),
			humanBytes(s.SizeBytes),
		})
	}

	ui.Println(ui.Bold("By Project"))
	fmt.Print(ui.RenderTable(headers, rows))
}

func summarizeProjects(items []project.Item) []ProjectSummary {
	byProj := map[string]*ProjectSummary{}
	for _, it := range items {
		key := it.Project
		if key == "" {
			key = "(unknown)"
		}
		a := byProj[key]
		if a == nil {
			a = &ProjectSummary{Project: key}
			byProj[key] = a
		}
		a.Items++
		a.SizeBytes += it.SizeBytes
		a.FileCount += it.FileCount
	}

	projects := make([]string, 0, len(byProj))
	for k := range byProj {
		projects = append(projects, k)
	}
	sort.Strings(projects)

	out := make([]ProjectSummary, 0, len(projects))
	for _, p := range projects {
		out = append(out, *byProj[p])
	}
	return out
}

func printProjectCleanTable(results []project.CleanResult, output string, pathMax int) {
	if len(results) == 0 {
		return
	}
	headers := []string{"Item", "Category", "Status", "Path"}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		status := r.Status
		switch r.Status {
		case "deleted":
			status = ui.Green("Deleted")
		case "dry-run":
			status = ui.Yellow("Dry-Run")
		case "skipped":
			status = ui.Yellow("Skipped")
		case "partial":
			status = ui.Yellow("Partial")
		case "error":
			status = ui.Red("Error")
		}
		if r.Error != "" {
			status = status + " (" + r.Error + ")"
		}
		rows = append(rows, []string{r.Item.Name, r.Item.Category, status, truncate(r.Item.Path, pathMax)})
	}

	switch output {
	case "json":
		printJSON(struct {
			Results []project.CleanResult `json:"results"`
		}{Results: results})
	case "csv":
		printCSV(headers, rows)
	default:
		fmt.Print(ui.RenderTable(headers, rows))
	}
}

func renderCacheStatus(it cache.Item) string {
	switch it.Status {
	case "ok":
		return ui.Green("OK")
	case "missing":
		return ui.Yellow("Missing")
	case "partial":
		return ui.Yellow("Partial")
	case "error":
		return ui.Red("Error")
	default:
		if it.Kind == cache.ItemDocker {
			return ui.Yellow(it.Status)
		}
		return it.Status
	}
}

func formatSize(it cache.Item) string {
	if it.SizeText != "" {
		return it.SizeText
	}
	if it.Kind == cache.ItemDocker {
		return "-"
	}
	if it.SizeBytes < 0 {
		return "-"
	}
	return humanBytes(it.SizeBytes)
}

func formatCount(count int64, kind cache.ItemKind) string {
	if kind == cache.ItemDocker {
		return "-"
	}
	if count < 0 {
		return "-"
	}
	return fmt.Sprintf("%d", count)
}

func humanBytes(v int64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div := int64(unit)
	exp := 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	pre := "KMGTPE"[exp]
	return fmt.Sprintf("%.1f %cB", float64(v)/float64(div), pre)
}

func printCleanTable(results []cache.CleanResult, output string, pathMax int) {
	headers := []string{"Item", "Category", "Status", "Path"}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		status := r.Status
		switch r.Status {
		case "deleted":
			status = ui.Green("Deleted")
		case "dry-run":
			status = ui.Yellow("Dry-Run")
		case "missing":
			status = ui.Yellow("Missing")
		case "skipped":
			status = ui.Yellow("Skipped")
		case "partial":
			status = ui.Yellow("Partial")
		case "error":
			status = ui.Red("Error")
		}
		if r.Error != "" {
			status = status + " (" + r.Error + ")"
		}
		rows = append(rows, []string{r.Item.Name, r.Item.Category, status, truncate(r.Item.Path, pathMax)})
	}

	switch output {
	case "json":
		printJSON(struct {
			Results []cache.CleanResult `json:"results"`
		}{Results: results})
	case "csv":
		printCSV(headers, rows)
	default:
		fmt.Print(ui.RenderTable(headers, rows))
	}
}

func summarize(items []cache.Item) Summary {
	var totalSize int64
	var totalFiles int64
	for _, it := range items {
		if it.SizeBytes > 0 {
			totalSize += it.SizeBytes
		}
		if it.FileCount > 0 {
			totalFiles += it.FileCount
		}
	}
	return Summary{
		ItemCount: len(items),
		FileCount: totalFiles,
		SizeBytes: totalSize,
	}
}

func printSummary(items []cache.Item) {
	s := summarize(items)
	headers := []string{"Items", "Files", "Size"}
	rows := [][]string{{
		fmt.Sprintf("%d", s.ItemCount),
		fmt.Sprintf("%d", s.FileCount),
		humanBytes(s.SizeBytes),
	}}
	ui.Println(ui.Bold("Summary"))
	fmt.Print(ui.RenderTable(headers, rows))
}

func printCategorySummary(items []cache.Item) {
	type agg struct {
		SizeBytes int64
		FileCount int64
		Items     int
	}
	byCat := map[string]*agg{}
	for _, it := range items {
		a := byCat[it.Category]
		if a == nil {
			a = &agg{}
			byCat[it.Category] = a
		}
		a.Items++
		if it.SizeBytes > 0 {
			a.SizeBytes += it.SizeBytes
		}
		if it.FileCount > 0 {
			a.FileCount += it.FileCount
		}
	}

	cats := make([]string, 0, len(byCat))
	for k := range byCat {
		cats = append(cats, k)
	}
	sort.Strings(cats)

	headers := []string{"Category", "Items", "Files", "Size"}
	rows := make([][]string, 0, len(cats))
	for _, c := range cats {
		a := byCat[c]
		rows = append(rows, []string{
			c,
			fmt.Sprintf("%d", a.Items),
			fmt.Sprintf("%d", a.FileCount),
			humanBytes(a.SizeBytes),
		})
	}

	ui.Println(ui.Bold("By Category"))
	fmt.Print(ui.RenderTable(headers, rows))
}

func printTopItems(items []cache.Item) {
	type entry struct {
		Name string
		Size int64
		Path string
	}
	list := make([]entry, 0, len(items))
	for _, it := range items {
		if it.SizeBytes > 0 {
			list = append(list, entry{Name: it.Name, Size: it.SizeBytes, Path: it.Path})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Size > list[j].Size })
	if len(list) == 0 {
		return
	}
	if len(list) > 5 {
		list = list[:5]
	}
	headers := []string{"Item", "Size", "Path"}
	rows := make([][]string, 0, len(list))
	for _, e := range list {
		rows = append(rows, []string{e.Name, humanBytes(e.Size), e.Path})
	}
	ui.Println(ui.Bold("Top Items"))
	fmt.Print(ui.RenderTable(headers, rows))
}

func scanProjectItems(root string, depth int, projectExclude string, projectNoDefault bool) []project.Item {
	if root == "" {
		return nil
	}
	exclude := []string{}
	if !projectNoDefault {
		exclude = append(exclude, defaultProjectExcludes()...)
	}
	if projectExclude != "" {
		parts := strings.Split(projectExclude, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				exclude = append(exclude, p)
			}
		}
	}
	targets, err := project.Scan(project.Options{
		Root:     root,
		MaxDepth: depth,
		Exclude:  exclude,
	})
	if err != nil {
		return nil
	}
	items := project.TargetsToItems(targets)
	return project.Stat(items)
}

func buildSizeMap(cacheItems []cache.Item, projectItems []project.Item) map[string]int64 {
	m := map[string]int64{}
	for _, it := range cacheItems {
		if it.Path != "" && it.SizeBytes > 0 {
			m[strings.ToLower(it.Path)] = it.SizeBytes
		}
	}
	for _, it := range projectItems {
		if it.Path != "" && it.SizeBytes > 0 {
			m[strings.ToLower(it.Path)] = it.SizeBytes
		}
	}
	return m
}

func deletedBytes(results []cache.CleanResult, sizeByPath map[string]int64) int64 {
	var total int64
	for _, r := range results {
		if r.DeletedBytes > 0 {
			total += r.DeletedBytes
			continue
		}
		if r.Status == "deleted" {
			if sz, ok := sizeByPath[strings.ToLower(r.Item.Path)]; ok {
				total += sz
			}
		}
	}
	return total
}

func deletedProjectBytes(results []project.CleanResult, sizeByPath map[string]int64) int64 {
	var total int64
	for _, r := range results {
		if r.Status == "deleted" {
			if sz, ok := sizeByPath[strings.ToLower(r.Item.Path)]; ok {
				total += sz
			}
		}
	}
	return total
}

func defaultProjectExcludes() []string {
	return []string{
		".git",
		".svn",
		".hg",
		".idea",
		".vscode",
		".vs",
		".node_modules",
	}
}

func filterItems(items []cache.Item, minMB int64, minFiles int64) []cache.Item {
	if minMB <= 0 && minFiles <= 0 {
		return items
	}
	minBytes := minMB * 1024 * 1024
	out := make([]cache.Item, 0, len(items))
	for _, it := range items {
		if it.Kind == cache.ItemDocker {
			out = append(out, it)
			continue
		}
		if it.Status == "missing" || it.Status == "error" {
			continue
		}
		if minBytes > 0 && it.SizeBytes >= 0 && it.SizeBytes < minBytes {
			continue
		}
		if minFiles > 0 && it.FileCount >= 0 && it.FileCount < minFiles {
			continue
		}
		out = append(out, it)
	}
	return out
}

func filterRecycleBin(items []cache.Item) []cache.Item {
	out := make([]cache.Item, 0, len(items))
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Path), "\\$recycle.bin") {
			out = append(out, it)
		}
	}
	return out
}

func parseDockerReclaimed(res cache.CleanResult) int64 {
	if res.Error == "" {
		return 0
	}
	line := strings.ToLower(res.Error)
	if idx := strings.Index(line, "total reclaimed space"); idx >= 0 {
		parts := strings.Fields(res.Error)
		for i := 0; i < len(parts)-1; i++ {
			if strings.ToLower(parts[i]) == "space:" {
				if b, ok := cache.ParseSizeString(parts[i+1]); ok {
					return b
				}
			}
		}
	}
	if b, ok := cache.ParseSizeString(res.Error); ok {
		return b
	}
	return 0
}

func shouldWarnDiff(estimate int64, diff int64) bool {
	if estimate <= 0 {
		return false
	}
	const minWarn = int64(100 * 1024 * 1024)
	if diff < minWarn {
		return false
	}
	ratio := float64(diff) / float64(estimate)
	return ratio >= 0.15
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func loadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func saveConfigFile(path string, cfg Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func currentConfig(noColor bool, timeout time.Duration, showMissing bool, includeSystem bool, apply bool, dockerPrune bool, dockerAll bool, dockerVolumes bool, allowSystemDelete bool, minMB int64, minFiles int64, projectRoot string, projectDepth int, projectClean bool, projectReview bool, projectExclude string, projectNoDefault bool, recycleBinOnly bool, output string, cmdMax int, pathMax int) Config {
	return Config{
		NoColor:           boolPtr(noColor),
		Timeout:           strPtr(timeout.String()),
		ShowMissing:       boolPtr(showMissing),
		IncludeSystem:     boolPtr(includeSystem),
		Apply:             boolPtr(apply),
		DockerPrune:       boolPtr(dockerPrune),
		DockerAll:         boolPtr(dockerAll),
		DockerVolumes:     boolPtr(dockerVolumes),
		AllowSystemDelete: boolPtr(allowSystemDelete),
		MinMB:             int64Ptr(minMB),
		MinFiles:          int64Ptr(minFiles),
		ProjectRoot:       strPtr(projectRoot),
		ProjectDepth:      intPtr(projectDepth),
		ProjectClean:      boolPtr(projectClean),
		ProjectReview:     boolPtr(projectReview),
		ProjectExclude:    strPtr(projectExclude),
		ProjectNoDefault:  boolPtr(projectNoDefault),
		RecycleBinOnly:    boolPtr(recycleBinOnly),
		Output:            strPtr(output),
		CmdMax:            intPtr(cmdMax),
		PathMax:           intPtr(pathMax),
	}
}

func applyConfig(cfg Config, set map[string]bool, noColor *bool, timeout *time.Duration, showMissing *bool, includeSystem *bool, apply *bool, dockerPrune *bool, dockerAll *bool, dockerVolumes *bool, allowSystemDelete *bool, minMB *int64, minFiles *int64, projectRoot *string, projectDepth *int, projectClean *bool, projectReview *bool, projectExclude *string, projectNoDefault *bool, recycleBinOnly *bool, output *string, cmdMax *int, pathMax *int) {
	if cfg.NoColor != nil && !set["no-color"] {
		*noColor = *cfg.NoColor
	}
	if cfg.Timeout != nil && !set["timeout"] {
		if d, err := time.ParseDuration(*cfg.Timeout); err == nil {
			*timeout = d
		}
	}
	if cfg.ShowMissing != nil && !set["show-missing"] {
		*showMissing = *cfg.ShowMissing
	}
	if cfg.IncludeSystem != nil && !set["include-system"] {
		*includeSystem = *cfg.IncludeSystem
	}
	if cfg.Apply != nil && !set["apply"] {
		*apply = *cfg.Apply
	}
	if cfg.DockerPrune != nil && !set["docker-prune"] {
		*dockerPrune = *cfg.DockerPrune
	}
	if cfg.DockerAll != nil && !set["docker-all"] {
		*dockerAll = *cfg.DockerAll
	}
	if cfg.DockerVolumes != nil && !set["docker-volumes"] {
		*dockerVolumes = *cfg.DockerVolumes
	}
	if cfg.AllowSystemDelete != nil && !set["allow-system-delete"] {
		*allowSystemDelete = *cfg.AllowSystemDelete
	}
	if cfg.MinMB != nil && !set["min-mb"] {
		*minMB = *cfg.MinMB
	}
	if cfg.MinFiles != nil && !set["min-files"] {
		*minFiles = *cfg.MinFiles
	}
	if cfg.ProjectRoot != nil && !set["project-root"] {
		*projectRoot = *cfg.ProjectRoot
	}
	if cfg.ProjectDepth != nil && !set["project-depth"] {
		*projectDepth = *cfg.ProjectDepth
	}
	if cfg.ProjectClean != nil && !set["project-clean"] {
		*projectClean = *cfg.ProjectClean
	}
	if cfg.ProjectReview != nil && !set["project-review"] {
		*projectReview = *cfg.ProjectReview
	}
	if cfg.ProjectExclude != nil && !set["project-exclude"] {
		*projectExclude = *cfg.ProjectExclude
	}
	if cfg.ProjectNoDefault != nil && !set["project-no-default-exclude"] {
		*projectNoDefault = *cfg.ProjectNoDefault
	}
	if cfg.RecycleBinOnly != nil && !set["recycle-bin-only"] {
		*recycleBinOnly = *cfg.RecycleBinOnly
	}
	if cfg.Output != nil && !set["output"] {
		*output = *cfg.Output
	}
	if cfg.CmdMax != nil && !set["cmd-max"] {
		*cmdMax = *cfg.CmdMax
	}
	if cfg.PathMax != nil && !set["path-max"] {
		*pathMax = *cfg.PathMax
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func strPtr(v string) *string {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func printCSV(headers []string, rows [][]string) {
	w := csv.NewWriter(os.Stdout)
	_ = w.Write(headers)
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()
}
