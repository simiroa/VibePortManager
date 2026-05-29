//ff:what §2.6 Smart Port Auto-Reassign — 가장 가까운 빈 포트 찾기
//ff:why 충돌 시 사용자가 수동으로 포트를 찾지 않아도 되게
package ports

import (
	"fmt"
	"math/rand"
	"net"

	vmsys "github.com/user/vpm/pkg/syscall"
)

const (
	suggestScanRange   = 50
	ephemeralPortStart = 49152
	ephemeralPortEnd   = 65535
)

// SuggestFree returns the nearest available port starting from startFrom+1.
// It tries up to suggestScanRange consecutive ports; if none are free it falls
// back to a random ephemeral port.
func SuggestFree(b vmsys.Backend, startFrom int) (int, error) {
	for delta := 1; delta <= suggestScanRange; delta++ {
		port := startFrom + delta
		if port > 65535 {
			break
		}
		if isFree(b, port) {
			return port, nil
		}
	}
	// Fallback: random ephemeral port.
	for attempts := 0; attempts < 20; attempts++ {
		port := ephemeralPortStart + rand.Intn(ephemeralPortEnd-ephemeralPortStart+1)
		if isFree(b, port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found near %d", startFrom)
}

// isFree returns true if nothing is listening on port.
// Uses net.Listen as the primary check; falls back to ResolvePortOwner.
func isFree(b vmsys.Backend, port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err == nil {
		_ = ln.Close()
		return true
	}
	owner, err := b.ResolvePortOwner(port)
	if err != nil {
		return false
	}
	return owner.PID == 0
}
