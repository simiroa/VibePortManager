//ff:what Procfile(foreman/honcho/Heroku) 프로세스들을 후보로 추출
//ff:why 언어 무관 범용 포맷. 포트는 보통 $PORT라 미상(사용자 입력)
package project

import (
	"os"
	"path/filepath"
	"strings"
)

type procfileDetector struct{}

func (procfileDetector) Name() string { return "procfile" }

func (procfileDetector) Detect(dir string) []DetectedServer {
	data, err := os.ReadFile(filepath.Join(dir, "Procfile"))
	if err != nil {
		return nil
	}
	var out []DetectedServer
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, cmd, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name, cmd = strings.TrimSpace(name), strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}
		out = append(out, DetectedServer{Name: name, Command: cmd, Port: 0, Source: "procfile"})
	}
	return out
}
