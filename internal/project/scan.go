package project

import (
	"os"
	"path/filepath"
	"strings"
)

type Rule struct {
	Name     string
	Patterns []string
	Category string
}

type Options struct {
	Root       string
	Exclude    []string
	MaxDepth   int
	ShowHidden bool
}

type Target struct {
	Name     string
	Path     string
	Category string
	Project  string
}

func DefaultRules() []Rule {
	return []Rule{
		rule("node_modules", "Web", "node_modules"),
		rule(".gradle", "Java", ".gradle"),
		rule(".mvn", "Java", ".mvn"),
		rule("target", "Java", "target"),
		rule("build", "Build", "build"),
		rule("dist", "Build", "dist"),
		rule(".next", "Web", ".next"),
		rule(".nuxt", "Web", ".nuxt"),
		rule(".cache", "Build", ".cache"),
		rule(".venv", "Python", ".venv"),
		rule("venv", "Python", "venv"),
		rule("__pycache__", "Python", "__pycache__"),
		rule(".pytest_cache", "Python", ".pytest_cache"),
		rule(".mypy_cache", "Python", ".mypy_cache"),
		rule(".ruff_cache", "Python", ".ruff_cache"),
		rule(".tox", "Python", ".tox"),
		rule(".idea", "Editor", ".idea"),
		rule(".vscode", "Editor", ".vscode"),
		rule(".vs", "Editor", ".vs"),
		rule(".sln", "Editor", "*.sln"),
		rule(".DS_Store", "OS", ".DS_Store"),
	}
}

func rule(name, category string, patterns ...string) Rule {
	return Rule{Name: name, Category: category, Patterns: patterns}
}

func Scan(opts Options) ([]Target, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	exclude := make([]string, 0, len(opts.Exclude))
	for _, e := range opts.Exclude {
		exclude = append(exclude, strings.ToLower(filepath.Clean(e)))
	}

	maxDepth := opts.MaxDepth
	if maxDepth == 0 {
		maxDepth = 5
	}
	if maxDepth < 0 {
		maxDepth = int(^uint(0) >> 1) // effectively unlimited
	}
	if maxDepth <= 0 {
		maxDepth = 5
	}

	var targets []Target
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		depth := strings.Count(rel, string(os.PathSeparator)) + 1
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if !opts.ShowHidden && strings.HasPrefix(name, ".") {
			// allow known hidden cache names that we explicitly target
			if !isExplicitHiddenTarget(name) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		lowerPath := strings.ToLower(filepath.Clean(path))
		for _, ex := range exclude {
			if ex != "" && strings.HasPrefix(lowerPath, ex) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		for _, r := range DefaultRules() {
			if matchAny(path, r.Patterns) {
				project := projectRoot(path)
				targets = append(targets, Target{Name: r.Name, Path: path, Category: r.Category, Project: project})
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return targets, nil
}

func matchAny(path string, patterns []string) bool {
	base := filepath.Base(path)
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?") {
			if ok, _ := filepath.Match(p, base); ok {
				return true
			}
			continue
		}
		if strings.EqualFold(base, p) {
			return true
		}
	}
	return false
}

func isExplicitHiddenTarget(name string) bool {
	switch strings.ToLower(name) {
	case ".gradle", ".mvn", ".next", ".nuxt", ".cache", ".venv", ".pytest_cache", ".mypy_cache", ".ruff_cache", ".tox", ".idea", ".vscode", ".vs":
		return true
	default:
		return false
	}
}

func projectRoot(path string) string {
	dir := filepath.Dir(path)
	for {
		if hasProjectMarker(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

func hasProjectMarker(dir string) bool {
	markers := []string{".git", "package.json", "go.mod", "pom.xml", "build.gradle", "pyproject.toml", "requirements.txt"}
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
			return true
		}
	}
	return false
}
