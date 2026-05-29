//ff:what Triple-Pass Port Killer 전체 파이프라인 orchestrator
//ff:why Graceful→Poll→(Released|Force)→Result 시퀀스 조립
package portkiller

import (
	"context"
	"fmt"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// Run executes the full Triple-Pass sequence against port for a running handle.
// allBackends is used for cross-target detection in Force phase.
// Returns the final Result which maps directly to a UI collision event payload.
func Run(ctx context.Context, primary vmsys.Backend, allBackends []vmsys.Backend, handle vmsys.Handle, port int) (Result, error) {
	// Phase 1 — Graceful signal.
	if err := Graceful(primary, handle); err != nil {
		// Non-fatal: process may already be dead. Proceed to polling.
		_ = err
	}

	// Phase 2 — Poll 3×500ms.
	released, err := Poll(ctx, primary, port)
	if err != nil {
		return Result{}, fmt.Errorf("poll phase: %w", err)
	}
	if released {
		return Result{FinalState: Released}, nil
	}

	// Phase 3 — Resolve and force.
	result, err := Force(primary, allBackends, port)
	if err != nil {
		return result, fmt.Errorf("force phase: %w", err)
	}
	return result, nil
}
