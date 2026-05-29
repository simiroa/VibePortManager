package logbuf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogDir_NoAPPDATA(t *testing.T) {
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", "")
	defer os.Setenv("APPDATA", orig)

	_, err := LogDir("proj1")
	if err == nil {
		t.Error("expected error when APPDATA not set")
	}
}

func TestLogDir_Creates(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("APPDATA", tmp)
	defer os.Unsetenv("APPDATA")

	dir, err := LogDir("proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("directory not created")
	}
}

func TestNewLogFile_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("APPDATA", tmp)
	defer os.Unsetenv("APPDATA")

	f, path, err := NewLogFile("proj1", "server1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("log file not created")
	}
}

func TestRotate_DeletesOldFiles(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("APPDATA", tmp)
	defer os.Unsetenv("APPDATA")

	dir, _ := LogDir("proj1")

	// Create an old file (8 days ago).
	old := filepath.Join(dir, "old.log")
	os.WriteFile(old, []byte("x"), 0644)
	oldTime := time.Now().AddDate(0, 0, -8)
	os.Chtimes(old, oldTime, oldTime)

	// Create a recent file.
	recent := filepath.Join(dir, "recent.log")
	os.WriteFile(recent, []byte("y"), 0644)

	if err := Rotate("proj1"); err != nil {
		t.Fatalf("Rotate error: %v", err)
	}

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old file should be deleted")
	}
	if _, err := os.Stat(recent); os.IsNotExist(err) {
		t.Error("recent file should remain")
	}
}

func TestRotate_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("APPDATA", tmp)
	defer os.Unsetenv("APPDATA")

	if err := Rotate("empty-proj"); err != nil {
		t.Fatalf("Rotate on empty dir: %v", err)
	}
}

func TestRotate_SkipsSubdirectories(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("APPDATA", tmp)
	defer os.Unsetenv("APPDATA")

	dir, _ := LogDir("proj-sub")
	// Create a subdirectory inside the log dir — Rotate must skip it.
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)
	// Create a regular log file.
	os.WriteFile(filepath.Join(dir, "run.log"), []byte("data"), 0644)

	if err := Rotate("proj-sub"); err != nil {
		t.Fatalf("Rotate error: %v", err)
	}
	// Subdir should still exist.
	if _, err := os.Stat(filepath.Join(dir, "subdir")); os.IsNotExist(err) {
		t.Error("subdir should not be deleted by Rotate")
	}
}

func TestRotate_MultipleRecentFiles(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("APPDATA", tmp)
	defer os.Unsetenv("APPDATA")

	dir, _ := LogDir("proj-multi")
	for _, name := range []string{"run0.log", "run1.log", "run2.log"} {
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644)
	}

	if err := Rotate("proj-multi"); err != nil {
		t.Fatalf("Rotate error: %v", err)
	}
}
