//ff:what 범용 태스크러너의 dev 타겟 탐지 (Makefile / justfile / Taskfile)
//ff:why 폴리글랏·비-JS 프로젝트가 흔히 쓰는 진입점. 포트 미상(사용자 입력)
package project

import (
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

type taskRunnerDetector struct{}

func (taskRunnerDetector) Name() string { return "task" }

// reTarget matches a Makefile/justfile recipe header: `name:` at line start.
var reTarget = regexp.MustCompile(`(?m)^([A-Za-z0-9_-]+)\s*:`)

func (taskRunnerDetector) Detect(dir string) []DetectedServer {
	var out []DetectedServer

	// Makefile targets.
	if data := firstFile(dir, "Makefile", "makefile", "GNUmakefile"); data != "" {
		for _, tgt := range targetNames(data) {
			if isDevish(tgt) {
				out = append(out, DetectedServer{Name: "make:" + tgt, Command: "make " + tgt, Source: "task"})
			}
		}
	}

	// justfile recipes.
	if data := firstFile(dir, "justfile", "Justfile", ".justfile"); data != "" {
		for _, r := range targetNames(data) {
			if isDevish(r) {
				out = append(out, DetectedServer{Name: "just:" + r, Command: "just " + r, Source: "task"})
			}
		}
	}

	// Taskfile (go-task).
	if raw, err := os.ReadFile(filepath.Join(dir, taskfileName(dir))); err == nil && len(raw) > 0 {
		var doc struct {
			Tasks map[string]yaml.Node `yaml:"tasks"`
		}
		if yaml.Unmarshal(raw, &doc) == nil {
			for name := range doc.Tasks {
				if isDevish(name) {
					out = append(out, DetectedServer{Name: "task:" + name, Command: "task " + name, Source: "task"})
				}
			}
		}
	}
	return out
}

func firstFile(dir string, names ...string) string {
	for _, n := range names {
		if data, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
			return string(data)
		}
	}
	return ""
}

func taskfileName(dir string) string {
	for _, n := range []string{"Taskfile.yml", "Taskfile.yaml", "taskfile.yml", "taskfile.yaml"} {
		if exists(filepath.Join(dir, n)) {
			return n
		}
	}
	return "Taskfile.yml"
}

func targetNames(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range reTarget.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if name == "PHONY" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
