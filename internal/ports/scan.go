//ff:what 시스템 전체 LISTEN 포트 + 점유 PID 스냅샷
//ff:why Tab 2 [System Port Analyzer Panel] 데이터 소스
package ports

import (
	"fmt"
	"sync"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// PortEntry represents one listening port across any backend.
type PortEntry struct {
	Port        int    `json:"port"`
	PID         int    `json:"pid"`
	ProcessName string `json:"processName"`
	BackendID   string `json:"backendId"` // "windows-native" | "wsl:Ubuntu"
}

// ScanAll queries every provided backend for listening ports and merges results.
// Errors from individual backends are soft-failures (partial results returned).
func ScanAll(backends []vmsys.Backend) ([]PortEntry, error) {
	type result struct {
		entries []PortEntry
		err     error
	}

	ch := make(chan result, len(backends))
	var wg sync.WaitGroup

	for _, b := range backends {
		wg.Add(1)
		go func(b vmsys.Backend) {
			defer wg.Done()
			raw, err := b.ScanListenPorts()
			if err != nil {
				ch <- result{err: err}
				return
			}
			entries := make([]PortEntry, len(raw))
			for i, e := range raw {
				entries[i] = PortEntry{
					Port:        e.Port,
					PID:         e.PID,
					ProcessName: e.Description,
					BackendID:   b.ID(),
				}
			}
			ch <- result{entries: entries}
		}(b)
	}

	wg.Wait()
	close(ch)

	var all []PortEntry
	var errs []error
	for r := range ch {
		if r.err != nil {
			errs = append(errs, r.err)
		}
		all = append(all, r.entries...)
	}

	if len(all) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("scan failed: %v", errs[0])
	}
	return all, nil
}
