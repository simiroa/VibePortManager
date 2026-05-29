//ff:what .NET 프로젝트 탐지 (*.csproj / *.sln → dotnet watch run)
//ff:why ASP.NET Core 등 .NET 백엔드 범용 대응. 포트는 launchSettings.json에서 추출(기본 5000)
package project

import (
	"path/filepath"
	"regexp"
	"strconv"
)

type dotnetDetector struct{}

func (dotnetDetector) Name() string { return "dotnet" }

var reURLPort = regexp.MustCompile(`https?://[^\s:/"]*:(\d{4,5})`)

func (dotnetDetector) Detect(dir string) []DetectedServer {
	has := func(pat string) bool {
		m, _ := filepath.Glob(filepath.Join(dir, pat))
		return len(m) > 0
	}
	if !has("*.csproj") && !has("*.sln") {
		return nil
	}
	port := 5000
	if data := readLower(dir, filepath.Join("properties", "launchsettings.json")); data != "" {
		if m := reURLPort.FindStringSubmatch(data); m != nil {
			if p, _ := strconv.Atoi(m[1]); p > 0 {
				port = p
			}
		}
	}
	return []DetectedServer{{
		Name:    filepath.Base(dir) + "-dotnet",
		Command: "dotnet watch run",
		Port:    port,
		Source:  "dotnet",
	}}
}
