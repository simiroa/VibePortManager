//ff:what Elixir Phoenix 탐지 (mix.exs + phoenix → mix phx.server)
//ff:why Phoenix 웹 백엔드 범용 대응. 기본 포트 4000
package project

import (
	"path/filepath"
	"strings"
)

type elixirDetector struct{}

func (elixirDetector) Name() string { return "elixir" }

func (elixirDetector) Detect(dir string) []DetectedServer {
	if !strings.Contains(readLower(dir, "mix.exs"), "phoenix") {
		return nil // non-Phoenix Elixir isn't a web server by default
	}
	return []DetectedServer{{
		Name:    filepath.Base(dir) + "-phoenix",
		Command: "mix phx.server",
		Port:    4000,
		Source:  "elixir",
	}}
}
