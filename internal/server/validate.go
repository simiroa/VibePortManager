//ff:what 포트 유일성 검증: 프로젝트 내 서버 간 중복 포트 거부
//ff:why 같은 포트로 두 서버 Start 시 즉시 PORT_COLLISION → 사전 차단
package server

import (
	"fmt"

	"github.com/user/vpm/internal/config"
)

// ValidatePortUnique checks that no other server in the same project uses the same port.
func ValidatePortUnique(proj config.Project, excludeServerID string, port int) error {
	for _, s := range proj.Servers {
		if s.ID == excludeServerID {
			continue
		}
		if s.Port == port {
			return fmt.Errorf("port %d already assigned to server %q", port, s.Name)
		}
	}
	return nil
}
