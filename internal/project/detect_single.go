//ff:what 단일 앱(루트 package.json에 dev 스크립트) 한 개를 후보로 추출
//ff:why 워크스페이스/compose가 있는 프로젝트에서도 루트 앱을 같이 잡기 위함
package project

type singleDetector struct{}

func (singleDetector) Name() string { return "package" }

func (singleDetector) Detect(dir string) []DetectedServer {
	// Workspace roots are handled by the workspace detector (children); skip here
	// so a monorepo root with a non-dev catch-all script isn't mis-registered.
	if len(readWorkspaces(dir)) > 0 {
		return nil
	}
	name, script := pickDevScript(dir)
	if script == "" {
		return nil // no dev-ish script → not a runnable single app
	}
	cmd := runScript(DetectPackageManager(dir), script)
	port := 0
	if p := detectPortStatic(dir, cmd); p != nil {
		port = *p
	}
	return []DetectedServer{{Name: name, Command: cmd, Port: port, Source: "package"}}
}
