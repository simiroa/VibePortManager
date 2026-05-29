package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "package.json"), `{"name":"mono","workspaces":["apps/*"]}`)
	write(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name":"web","scripts":{"dev":"vite"}}`)
	write(t, filepath.Join(dir, "apps", "web", ".env"), "PORT=4100\n")
	// a library workspace with no dev script → must be skipped
	write(t, filepath.Join(dir, "apps", "lib", "package.json"), `{"name":"lib","scripts":{"build":"tsc"}}`)

	got := workspaceDetector{}.Detect(dir)
	if len(got) != 1 {
		t.Fatalf("expected 1 app (lib skipped), got %d: %+v", len(got), got)
	}
	s := got[0]
	if s.Name != "web" || s.Port != 4100 {
		t.Errorf("got name=%q port=%d, want web/4100", s.Name, s.Port)
	}
	if !strings.Contains(s.Command, "run dev") || !strings.Contains(s.Command, "cd /d") {
		t.Errorf("command should cd into the workspace and run dev: %q", s.Command)
	}
}

func TestComposeDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "docker-compose.yml"), `
services:
  db:
    image: postgres
    ports:
      - "5432:5432"
  api:
    image: node
    ports:
      - "127.0.0.1:8000:8000"
  internal:
    image: redis
`)
	got := composeDetector{}.Detect(dir)
	ports := map[string]int{}
	for _, s := range got {
		ports[s.Name] = s.Port
	}
	if ports["db"] != 5432 {
		t.Errorf("db host port = %d, want 5432", ports["db"])
	}
	if ports["api"] != 8000 {
		t.Errorf("api host port = %d, want 8000 (ip:host:container)", ports["api"])
	}
	if _, ok := ports["internal"]; ok {
		t.Errorf("internal service has no published port and must be skipped")
	}
}

func TestSingleDetector(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "package.json"), `{"name":"app","scripts":{"dev":"vite"}}`)
	write(t, filepath.Join(dir, ".env"), "PORT=3000\n")

	got := singleDetector{}.Detect(dir)
	if len(got) != 1 || got[0].Port != 3000 || got[0].Source != "package" {
		t.Fatalf("single detector wrong: %+v", got)
	}
}

func TestSingleDetectorSkipsWorkspaceRoot(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "package.json"), `{"name":"mono","workspaces":["apps/*"],"scripts":{"dev":"echo root"}}`)
	if got := (singleDetector{}).Detect(dir); got != nil {
		t.Errorf("workspace root must not be a single app: %+v", got)
	}
}

func TestDetectAllDedupesByPort(t *testing.T) {
	dir := t.TempDir()
	// single app on 3000 + compose service also on 3000 → only first kept
	write(t, filepath.Join(dir, "package.json"), `{"name":"app","scripts":{"dev":"vite"}}`)
	write(t, filepath.Join(dir, ".env"), "PORT=3000\n")
	write(t, filepath.Join(dir, "docker-compose.yml"), "services:\n  web:\n    ports:\n      - \"3000:3000\"\n")

	got := DetectAll(dir)
	seen := map[int]int{}
	for _, s := range got {
		if s.Port > 0 {
			seen[s.Port]++
		}
	}
	if seen[3000] > 1 {
		t.Errorf("port 3000 should appear once after dedupe, got %d", seen[3000])
	}
}
