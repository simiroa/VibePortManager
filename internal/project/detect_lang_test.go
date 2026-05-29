package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonDjango(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "manage.py"), "# django")
	got := pythonDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Port != 8000 || !strings.Contains(got[0].Command, "runserver") {
		t.Fatalf("django detect wrong: %+v", got)
	}
}

func TestPythonFastAPI(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "requirements.txt"), "fastapi\nuvicorn[standard]\n")
	write(t, filepath.Join(dir, "app", "main.py"), "app = FastAPI()")
	got := pythonDetector{}.Detect(dir)
	if len(got) != 1 || !strings.Contains(got[0].Command, "uvicorn app.main:app") {
		t.Fatalf("fastapi detect wrong: %+v", got)
	}
}

func TestGoDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "go.mod"), "module x\n")
	write(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}")
	got := goDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Command != "go run ." {
		t.Fatalf("go detect wrong: %+v", got)
	}
}

func TestRustDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Cargo.toml"), "[package]\nname=\"x\"\n")
	got := rustDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Command != "cargo run" {
		t.Fatalf("rust detect wrong: %+v", got)
	}
}

func TestProcfileDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Procfile"), "web: gunicorn app:app\nworker: celery -A app worker\n# comment\n")
	got := procfileDetector{}.Detect(dir)
	if len(got) != 2 {
		t.Fatalf("expected 2 procs, got %d: %+v", len(got), got)
	}
}

func TestTaskRunnerMakefile(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Makefile"), ".PHONY: dev build\ndev:\n\tnpm run dev\nbuild:\n\ttsc\n")
	got := taskRunnerDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Command != "make dev" {
		t.Fatalf("expected only the dev target, got %+v", got)
	}
}

func TestFallbackOnlyWhenNoPrimary(t *testing.T) {
	// A repo with ONLY a Makefile (no package.json/python/go/rust) → fallback fires.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Makefile"), "serve:\n\t./run.sh\n")
	if got := DetectAll(dir); len(got) != 1 || got[0].Source != "task" {
		t.Fatalf("expected fallback task detector, got %+v", got)
	}

	// Add a package.json dev script → primary wins, Makefile suppressed.
	write(t, filepath.Join(dir, "package.json"), `{"name":"app","scripts":{"dev":"vite"}}`)
	got := DetectAll(dir)
	for _, s := range got {
		if s.Source == "task" {
			t.Errorf("task fallback should be suppressed when a primary detector matched: %+v", got)
		}
	}
}
