//ff:what 프로젝트 디렉토리에서 패키지 매니저 자동 감지
//ff:why 사용자 입력 최소화: lock file 존재 여부로 PM 결정
package project

import "os"

// DetectPackageManager inspects lockfiles in dir to identify the package manager.
// Priority: bun > pnpm > yarn > npm. Returns "none" if no lockfile found.
func DetectPackageManager(dir string) string {
	checks := []struct {
		file string
		pm   string
	}{
		{"bun.lockb", "bun"},
		{"bun.lock", "bun"}, // bun ≥1.1 text lockfile
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}
	for _, c := range checks {
		if _, err := os.Stat(dir + string(os.PathSeparator) + c.file); err == nil {
			return c.pm
		}
	}
	return "none"
}
