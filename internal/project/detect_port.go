//ff:what 디렉토리에서 dev 서버 포트를 정적으로 추출 (.env → 프레임워크 config → 명령 플래그)
//ff:why 워크스페이스·단일앱 탐지기가 공유하는 포트 추출 책임을 한 곳에 모음
package project

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// reEnvPort matches PORT=NNNN or VITE_PORT=NNNN in .env files.
	reEnvPort = regexp.MustCompile(`(?i)^(?:VITE_)?PORT\s*=\s*([0-9]{4,5})\s*$`)
	// reFilePort matches port: 3000 / port = 3000 / "port": 3000 in config files.
	reFilePort = regexp.MustCompile(`(?i)\bport\b[^0-9a-z_A-Z][\s:="']*([0-9]{4,5})\b`)
	// reCmdPort matches --port 3000 or --port=3000 in command strings.
	reCmdPort = regexp.MustCompile(`--port[=\s]+([0-9]{4,5})`)
)

// detectPortStatic extracts a port from env files, framework config, then the
// command's --port flag. Returns nil if none found.
func detectPortStatic(dir, cmd string) *int {
	// 1. .env files (highest specificity)
	for _, name := range []string{".env.local", ".env.development.local", ".env.development", ".env"} {
		if p := scanEnvFile(filepath.Join(dir, name)); p != nil {
			return p
		}
	}
	// 2. Framework config files
	for _, name := range []string{
		"vite.config.js", "vite.config.ts", "vite.config.mjs", "vite.config.mts",
		"next.config.js", "next.config.ts", "next.config.mjs",
		"nuxt.config.js", "nuxt.config.ts",
		"svelte.config.js",
	} {
		if p := scanPortFile(filepath.Join(dir, name)); p != nil {
			return p
		}
	}
	// 3. --port flag in the command string
	if cmd != "" {
		if m := reCmdPort.FindStringSubmatch(cmd); m != nil {
			if p, _ := strconv.Atoi(m[1]); p > 0 && p < 65536 {
				return &p
			}
		}
	}
	return nil
}

func scanEnvFile(path string) *int {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		if m := reEnvPort.FindStringSubmatch(line); m != nil {
			if p, _ := strconv.Atoi(m[1]); p > 0 && p < 65536 {
				return &p
			}
		}
	}
	return nil
}

func scanPortFile(path string) *int {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := reFilePort.FindStringSubmatch(sc.Text()); m != nil {
			if p, _ := strconv.Atoi(m[1]); p > 0 && p < 65536 {
				return &p
			}
		}
	}
	return nil
}
