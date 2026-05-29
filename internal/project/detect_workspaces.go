//ff:what 모노레포 워크스페이스(apps/*, packages/*)에서 dev 서버 앱들을 후보로 추출
//ff:why npm/yarn/pnpm/bun monorepo: 루트엔 dev 스크립트가 없고 각 앱 하위에 있음
package project

import (
	"os"
	"path/filepath"
)

type workspaceDetector struct{}

func (workspaceDetector) Name() string { return "workspace" }

func (workspaceDetector) Detect(dir string) []DetectedServer {
	globs := readWorkspaces(dir)
	if len(globs) == 0 {
		return nil
	}
	pm := DetectPackageManager(dir)

	var out []DetectedServer
	for _, glob := range globs {
		matches, _ := filepath.Glob(filepath.Join(dir, filepath.FromSlash(glob)))
		for _, wsDir := range matches {
			if fi, err := os.Stat(wsDir); err != nil || !fi.IsDir() {
				continue
			}
			name, script := pickDevScript(wsDir)
			if script == "" {
				continue // library package, not a runnable app
			}
			rel, err := filepath.Rel(dir, wsDir)
			if err != nil {
				continue
			}
			// VPM runs commands from the project root, so cd into the workspace first.
			cmd := "cd /d " + rel + " && " + runScript(pm, script)
			port := 0
			if p := detectPortStatic(wsDir, cmd); p != nil {
				port = *p
			}
			out = append(out, DetectedServer{Name: name, Command: cmd, Port: port, Source: "workspace"})
		}
	}
	return out
}
