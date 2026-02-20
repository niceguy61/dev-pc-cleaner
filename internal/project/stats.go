package project

import (
	"os"
	"path/filepath"
)

func TargetsToItems(targets []Target) []Item {
	items := make([]Item, 0, len(targets))
	for _, t := range targets {
		items = append(items, Item{
			Name:     t.Name,
			Category: t.Category,
			Path:     t.Path,
			Project:  t.Project,
			Kind:     ItemPath,
		})
	}
	return items
}

type ItemKind string

const (
	ItemPath ItemKind = "path"
)

type Item struct {
	Name      string
	Category  string
	Path      string
	Project   string
	SizeBytes int64
	FileCount int64
	Status    string
	Kind      ItemKind
}

func Stat(items []Item) []Item {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		info, err := os.Stat(it.Path)
		if err != nil {
			it.Status = "error"
			out = append(out, it)
			continue
		}
		if info.IsDir() {
			size, count := dirSize(it.Path)
			it.SizeBytes = size
			it.FileCount = count
			it.Status = "ok"
			out = append(out, it)
			continue
		}
		it.SizeBytes = info.Size()
		it.FileCount = 1
		it.Status = "ok"
		out = append(out, it)
	}
	return out
}

func dirSize(path string) (int64, int64) {
	var size int64
	var count int64
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
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
			return nil
		}
		size += fi.Size()
		count++
		return nil
	})
	return size, count
}
