//ff:what 프로젝트 삭제: 서버 중지 후 config에서 제거
//ff:why 서버 실행 중 삭제 시 orphan 프로세스 방지
package project

import (
	"fmt"

	"github.com/user/vpm/internal/config"
)

// Remove removes the project with the given ID from config and saves.
// The caller is responsible for stopping all running servers before calling Remove.
func Remove(cfg *config.Config, projectID string) error {
	idx := -1
	for i, p := range cfg.Projects {
		if p.ID == projectID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("project not found: %s", projectID)
	}
	cfg.Projects = append(cfg.Projects[:idx], cfg.Projects[idx+1:]...)
	return config.Save(cfg)
}
