//ff:what Rust 프로젝트 탐지 (Cargo.toml → cargo run)
//ff:why Rust 서버도 범용 대응. 포트는 보통 코드/env에 있어 미상(사용자 입력)
package project

import (
	"path/filepath"
)

type rustDetector struct{}

func (rustDetector) Name() string { return "rust" }

func (rustDetector) Detect(dir string) []DetectedServer {
	if !exists(filepath.Join(dir, "Cargo.toml")) {
		return nil
	}
	return []DetectedServer{{
		Name:    filepath.Base(dir),
		Command: "cargo run",
		Port:    0,
		Source:  "rust",
	}}
}
