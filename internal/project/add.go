//ff:what 프로젝트 등록: 경로 → execution_target 자동 감지 + PM 감지 + config 저장
//ff:why WSL UNC 경로(\\wsl$\...) → wsl 타겟, 그 외 → windows-native
package project

import (
	"fmt"
	"strings"

	"github.com/user/vpm/internal/config"
	"github.com/user/vpm/internal/wslwrap"
)

// AddResult carries the result of an Add operation.
type AddResult struct {
	Project         config.Project
	PMDetected      bool
	TargetDetected  bool
}

// Add registers a new project at the given path.
// It auto-detects execution_target and package_manager, then saves config.
// The caller must supply a pre-generated UUID for the project ID.
func Add(cfg *config.Config, id, name, path string) (AddResult, error) {
	target, distro := detectTarget(path)
	pm := DetectPackageManager(path)

	proj := config.Project{
		ID:              id,
		Name:            name,
		Path:            path,
		ExecutionTarget: target,
		WSLDistro:       distro,
		PackageManager:  pm,
		Servers:         []config.Server{},
	}

	for _, p := range cfg.Projects {
		if p.Path == path {
			return AddResult{}, fmt.Errorf("project already registered: %s", path)
		}
	}

	cfg.Projects = append(cfg.Projects, proj)
	if err := config.Save(cfg); err != nil {
		return AddResult{}, err
	}

	return AddResult{
		Project:        proj,
		PMDetected:     pm != "none",
		TargetDetected: true,
	}, nil
}

// detectTarget infers execution_target and wsl_distro from the path.
func detectTarget(path string) (target, distro string) {
	// \\wsl$\<Distro>\... or \\wsl.localhost\<Distro>\...
	for _, prefix := range []string{`\\wsl$\`, `\\wsl.localhost\`} {
		if strings.HasPrefix(path, prefix) {
			rest := path[len(prefix):]
			parts := strings.SplitN(rest, `\`, 2)
			if len(parts) >= 1 && parts[0] != "" {
				return "wsl", parts[0]
			}
		}
	}
	// Try matching a known distro name in a UNC path (fallback)
	_ = wslwrap.DetectNetworkMode // just ensure import is used
	return "windows-native", ""
}
