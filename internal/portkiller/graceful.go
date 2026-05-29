//ff:what Phase 1: 프로세스 트리에 graceful signal
//ff:why 자식까지 끄려면 트리/그룹 단위 시그널 필요
package portkiller

import (
	"fmt"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// Graceful sends a SIGTERM (or equivalent) to the process tree identified by
// handle. Transitions state from GracefulSent.
func Graceful(b vmsys.Backend, h vmsys.Handle) error {
	if err := b.SignalTree(h, vmsys.SignalTerm); err != nil {
		return fmt.Errorf("graceful signal: %w", err)
	}
	return nil
}
