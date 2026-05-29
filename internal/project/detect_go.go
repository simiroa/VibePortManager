//ff:what Go 프로젝트 탐지 (go.mod + main 패키지 → go run)
//ff:why Go 백엔드/서버도 범용 대응. 포트는 보통 코드/env에 있어 미상(사용자 입력)
package project

import (
	"path/filepath"
)

type goDetector struct{}

func (goDetector) Name() string { return "go" }

func (goDetector) Detect(dir string) []DetectedServer {
	if !exists(filepath.Join(dir, "go.mod")) {
		return nil
	}
	base := filepath.Base(dir)

	// Root main package.
	if exists(filepath.Join(dir, "main.go")) {
		return []DetectedServer{{Name: base, Command: "go run .", Port: 0, Source: "go"}}
	}
	// cmd/<name>/main.go layout.
	if matches, _ := filepath.Glob(filepath.Join(dir, "cmd", "*", "main.go")); len(matches) > 0 {
		cmdName := filepath.Base(filepath.Dir(matches[0]))
		return []DetectedServer{{
			Name:    base + "-" + cmdName,
			Command: "go run ./cmd/" + cmdName,
			Port:    0,
			Source:  "go",
		}}
	}
	return nil
}
