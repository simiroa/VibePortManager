package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTailFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(p, []byte("a\nb\nc\nd\ne\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := tailFile(p, 2)
	if err != nil {
		t.Fatalf("tailFile: %v", err)
	}
	if len(got) != 2 || got[0] != "d" || got[1] != "e" {
		t.Errorf("tail(2) = %v, want [d e]", got)
	}

	all, err := tailFile(p, 100)
	if err != nil {
		t.Fatalf("tailFile: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("tail(100) returned %d lines, want 5", len(all))
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "out", "dst.txt")
	content := []byte("hello\nworld\n")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	if err := copyFile(filepath.Join(dir, "nope.txt"), filepath.Join(dir, "out.txt")); err == nil {
		t.Errorf("expected error copying a missing source")
	}
}
