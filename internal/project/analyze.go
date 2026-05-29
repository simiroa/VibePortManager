//ff:what 프로젝트 정적 분석 진입점 — 단일 dev 명령 + 다중 서버 탐지 집계
//ff:why 자동 감지로 사용자 입력 최소화. 포트탐지는 detect_port.go, 다중탐지는 Detector 프레임워크
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ProjectAnalysis is the result of static analysis on a project directory.
// Port is nil when no port could be detected; the caller should enter Detection Mode.
// Servers holds multi-service candidates (monorepo workspaces, PM2 ecosystem,
// docker-compose, …); empty for a plain single-app project.
type ProjectAnalysis struct {
	Port       *int             `json:"port"`
	Command    string           `json:"command"`
	ScriptName string           `json:"scriptName"`
	PackageMgr string           `json:"packageMgr"`
	Servers    []DetectedServer `json:"servers"`
}

// Analyze performs static analysis on dir.
func Analyze(dir string) ProjectAnalysis {
	pm := DetectPackageManager(dir)
	scriptName, cmd := pickScript(dir, pm)
	return ProjectAnalysis{
		Port:       detectPortStatic(dir, cmd),
		Command:    cmd,
		ScriptName: scriptName,
		PackageMgr: pm,
		Servers:    DetectAll(dir),
	}
}

// pickScript returns the best single dev script (dev > start > serve > preview,
// then the first script as a last resort) and its run command. Used for the
// single-app proposal; multi-service detection lives in the Detector framework.
func pickScript(dir, pm string) (name, cmd string) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", ""
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil || len(pkg.Scripts) == 0 {
		return "", ""
	}
	for _, sc := range []string{"dev", "start", "serve", "preview"} {
		if _, ok := pkg.Scripts[sc]; ok {
			return sc, runScript(pm, sc)
		}
	}
	for sc := range pkg.Scripts {
		return sc, runScript(pm, sc)
	}
	return "", ""
}
