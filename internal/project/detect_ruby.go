//ff:what Ruby 웹 탐지 (Rails bin/dev·bin/rails / Rack config.ru)
//ff:why Rails 등 Ruby 백엔드 범용 대응
package project

import (
	"path/filepath"
	"strings"
)

type rubyDetector struct{}

func (rubyDetector) Name() string { return "ruby" }

func (rubyDetector) Detect(dir string) []DetectedServer {
	base := filepath.Base(dir)
	isRails := exists(filepath.Join(dir, "bin", "rails")) || strings.Contains(readLower(dir, "Gemfile"), "rails")

	switch {
	case isRails && exists(filepath.Join(dir, "bin", "dev")): // Rails 7 default (foreman)
		return []DetectedServer{{Name: base + "-rails", Command: "bin/dev", Port: 3000, Source: "ruby"}}
	case isRails:
		return []DetectedServer{{Name: base + "-rails", Command: "bin/rails server -p 3000", Port: 3000, Source: "ruby"}}
	case exists(filepath.Join(dir, "config.ru")):
		return []DetectedServer{{Name: base + "-rack", Command: "rackup -p 9292", Port: 9292, Source: "ruby"}}
	}
	return nil
}
