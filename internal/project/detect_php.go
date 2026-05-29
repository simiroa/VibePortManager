//ff:what PHP 웹 탐지 (Laravel artisan / Symfony / 내장 서버)
//ff:why PHP 백엔드 범용 대응
package project

import (
	"path/filepath"
)

type phpDetector struct{}

func (phpDetector) Name() string { return "php" }

func (phpDetector) Detect(dir string) []DetectedServer {
	base := filepath.Base(dir)
	switch {
	case exists(filepath.Join(dir, "artisan")): // Laravel
		return []DetectedServer{{Name: base + "-laravel", Command: "php artisan serve --port=8000", Port: 8000, Source: "php"}}
	case exists(filepath.Join(dir, "bin", "console")): // Symfony
		return []DetectedServer{{Name: base + "-symfony", Command: "php -S localhost:8000 -t public", Port: 8000, Source: "php"}}
	case exists(filepath.Join(dir, "public", "index.php")):
		return []DetectedServer{{Name: base + "-php", Command: "php -S localhost:8000 -t public", Port: 8000, Source: "php"}}
	case exists(filepath.Join(dir, "index.php")):
		return []DetectedServer{{Name: base + "-php", Command: "php -S localhost:8000", Port: 8000, Source: "php"}}
	}
	return nil
}
