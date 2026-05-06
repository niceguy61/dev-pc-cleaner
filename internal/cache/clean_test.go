package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemovePathKeepsDirectoryRoot(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	nestedDir := filepath.Join(cacheDir, "nested")
	if err := os.MkdirAll(nestedDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "file.log"), []byte("log"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "nested.log"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := removePath(cacheDir, false); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("cache root should remain: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("cache root should remain a directory")
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("cache root should be empty, got %d entries", len(entries))
	}
}

func TestRemovePathDeletesFile(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(root, "tool.log")
	if err := os.WriteFile(logFile, []byte("log"), 0o600); err != nil {
		t.Fatal(err)
	}

	deletedBytes, err := removePath(logFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if deletedBytes != 3 {
		t.Fatalf("deleted bytes = %d, want 3", deletedBytes)
	}

	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Fatalf("log file should be deleted, got err=%v", err)
	}
}

func TestCleanReportsPartialDirectoryDelete(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	deletedFile := filepath.Join(cacheDir, "deleted.log")
	if err := os.WriteFile(deletedFile, []byte("deleted"), 0o600); err != nil {
		t.Fatal(err)
	}

	results := Clean([]Item{{
		Name:      "test cache",
		Category:  "Test",
		Path:      cacheDir,
		SizeBytes: 7,
		Status:    "ok",
		Kind:      ItemPath,
	}}, true, false)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Status != "deleted" {
		t.Fatalf("status = %q, want deleted", results[0].Status)
	}
	if results[0].DeletedBytes != 7 {
		t.Fatalf("deleted bytes = %d, want 7", results[0].DeletedBytes)
	}
}
