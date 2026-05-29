//ff:what Phase 2: 500ms x 3회 포트 TCP probe
//ff:why 양보 시간 부여. 정상 종료 케이스의 다수가 여기서 끝남
package portkiller

import (
	"context"
	"fmt"
	"net"
	"time"

	vmsys "github.com/user/vpm/pkg/syscall"
)

const (
	pollCount    = 3
	pollInterval = 500 * time.Millisecond
)

// Poll checks whether port has been released after a graceful signal.
// It probes pollCount times, each pollInterval apart.
// Returns (true, nil) if the port is free, (false, nil) if still occupied
// after all attempts, or (false, err) on an unexpected error.
func Poll(ctx context.Context, b vmsys.Backend, port int) (released bool, err error) {
	for i := 0; i < pollCount; i++ {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(pollInterval):
		}
		free, probeErr := probePort(b, port)
		if probeErr != nil {
			return false, fmt.Errorf("poll probe: %w", probeErr)
		}
		if free {
			return true, nil
		}
	}
	return false, nil
}

// probePort returns true if nothing is listening on port.
// Strategy (a): try net.Listen — cheapest accurate check.
// Strategy (b): Backend.ResolvePortOwner fallback.
func probePort(b vmsys.Backend, port int) (bool, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		_ = ln.Close()
		return true, nil
	}
	// Strategy (b): ask backend
	owner, err := b.ResolvePortOwner(port)
	if err != nil {
		return false, err
	}
	return owner.PID == 0, nil
}
