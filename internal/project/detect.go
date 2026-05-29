//ff:what 다중 서버 탐지 프레임워크 — 프로젝트 유형별 Detector 플러그인 + 집계
//ff:why 유형마다 파일 하나(책임 분리). 새 유형은 Detector 추가 후 detectors()에 등록만 하면 됨
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DetectedServer is one server candidate parsed from a project. Port may be 0
// when it cannot be inferred statically — the caller then asks the user for it
// (or VPM re-detects it after start).
type DetectedServer struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Port    int    `json:"port"`
	Source  string `json:"source"` // "package" | "workspace" | "pm2" | "compose"
}

// Detector inspects a project directory and returns server candidates for one
// project type. Implementations live in detect_<type>.go.
type Detector interface {
	Name() string
	Detect(dir string) []DetectedServer
}

// primaryDetectors recognise a concrete project type (earlier wins on port clash).
func primaryDetectors() []Detector {
	return []Detector{
		ecosystemDetector{}, // PM2 — explicit, highest fidelity
		composeDetector{},   // docker-compose services
		workspaceDetector{}, // npm/yarn/pnpm/bun monorepo apps
		singleDetector{},    // plain single-app dev script
		pythonDetector{},    // Django / FastAPI / Flask
		goDetector{},        // go run
		rustDetector{},      // cargo run
		dotnetDetector{},    // dotnet watch run
		rubyDetector{},      // Rails / Rack
		phpDetector{},       // Laravel / Symfony / built-in server
		javaDetector{},      // Spring Boot (Maven / Gradle)
		elixirDetector{},    // Phoenix
		denoDetector{},      // deno task
	}
}

// fallbackDetectors are generic task runners, used only when no primary detector
// matched (otherwise they'd duplicate the real dev command, e.g. a Makefile that
// just calls `npm run dev`).
func fallbackDetectors() []Detector {
	return []Detector{
		procfileDetector{},   // Procfile (foreman/honcho/Heroku)
		taskRunnerDetector{}, // Makefile / justfile / Taskfile dev targets
	}
}

// DetectAll runs the primary detectors; if none match, the fallbacks. Results
// are de-duplicated by port (first wins) and by display name.
func DetectAll(dir string) []DetectedServer {
	out := runDetectors(primaryDetectors(), dir)
	if len(out) == 0 {
		out = runDetectors(fallbackDetectors(), dir)
	}
	return out
}

func runDetectors(ds []Detector, dir string) []DetectedServer {
	var out []DetectedServer
	seenPort := map[int]bool{}
	seenName := map[string]bool{}
	for _, d := range ds {
		for _, s := range d.Detect(dir) {
			if s.Command == "" || seenName[s.Name] {
				continue
			}
			if s.Port > 0 && seenPort[s.Port] {
				continue
			}
			if s.Source == "" {
				s.Source = d.Name()
			}
			if s.Port > 0 {
				seenPort[s.Port] = true
			}
			seenName[s.Name] = true
			out = append(out, s)
		}
	}
	return out
}

// ── shared file/heuristic helpers ───────────────────────────────────────────

// exists reports whether a file or dir exists at path.
func exists(path string) bool { _, err := os.Stat(path); return err == nil }

// readLower returns the lower-cased contents of dir/name, or "" if absent.
func readLower(dir, name string) string {
	if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
		return strings.ToLower(string(data))
	}
	return ""
}

// venvBin returns a Windows venv executable path (.venv\Scripts\tool.exe) if it
// exists, otherwise the bare tool name (assumed on PATH).
func venvBin(dir, tool string) string {
	p := filepath.Join(".venv", "Scripts", tool+".exe")
	if exists(filepath.Join(dir, p)) {
		return p
	}
	return tool
}

// isDevish reports whether a task/target name looks like a dev-server command.
func isDevish(name string) bool {
	n := strings.ToLower(name)
	switch n {
	case "dev", "run", "serve", "start", "server", "web", "api", "watch", "develop":
		return true
	}
	return strings.Contains(n, "dev") || strings.Contains(n, "serve") || strings.Contains(n, "runserver")
}

// ── shared helpers ──────────────────────────────────────────────────────────

// runScript builds a package-manager run command (npm/pnpm/yarn/bun all accept
// the `<pm> run <script>` form).
func runScript(pm, script string) string {
	if pm == "" || pm == "none" {
		pm = "npm"
	}
	return pm + " run " + script
}

// pickDevScript reads dir/package.json and returns the package name (or dir base)
// and a dev-ish script (dev > start > serve > preview). script is "" when none
// exists — i.e. the package is a library, not a runnable app.
func pickDevScript(dir string) (name, script string) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", ""
	}
	var pkg struct {
		Name    string            `json:"name"`
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return "", ""
	}
	name = pkg.Name
	if name == "" {
		name = filepath.Base(dir)
	}
	for _, sc := range []string{"dev", "start", "serve", "preview"} {
		if _, ok := pkg.Scripts[sc]; ok {
			return name, sc
		}
	}
	return name, ""
}

// readWorkspaces returns the workspace globs declared by a monorepo root, from
// package.json "workspaces" (array or { packages: [...] }) or pnpm-workspace.yaml.
func readWorkspaces(dir string) []string {
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg struct {
			Workspaces json.RawMessage `json:"workspaces"`
		}
		if json.Unmarshal(data, &pkg) == nil && len(pkg.Workspaces) > 0 {
			var arr []string
			if json.Unmarshal(pkg.Workspaces, &arr) == nil && len(arr) > 0 {
				return arr
			}
			var obj struct {
				Packages []string `json:"packages"`
			}
			if json.Unmarshal(pkg.Workspaces, &obj) == nil && len(obj.Packages) > 0 {
				return obj.Packages
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "pnpm-workspace.yaml")); err == nil {
		var y struct {
			Packages []string `yaml:"packages"`
		}
		if yaml.Unmarshal(data, &y) == nil {
			return y.Packages
		}
	}
	return nil
}
