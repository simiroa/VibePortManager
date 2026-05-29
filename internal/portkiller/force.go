//ff:what Phase 3: 실제 점유 PID 해결 후 강제 종료
//ff:why graceful로 안 죽는 좀비를 물리적으로 해제
package portkiller

import (
	"fmt"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// Result is the outcome of the full Triple-Pass sequence.
type Result struct {
	FinalState State
	Killed     bool
	Owner      vmsys.PortOwner // zero if Released before Phase 3
}

// Force resolves who is blocking port and attempts a forceful kill.
// It consults allBackends to detect cross-target blockers.
// Returns a Result that the caller maps to a UI event.
func Force(primary vmsys.Backend, allBackends []vmsys.Backend, port int) (Result, error) {
	// Step 1: ask primary backend.
	owner, err := primary.ResolvePortOwner(port)
	if err != nil {
		return Result{}, fmt.Errorf("resolve primary: %w", err)
	}

	// Step 2: if not found locally, scan other backends.
	if owner.PID == 0 && len(allBackends) > 1 {
		for _, b := range allBackends {
			if b.ID() == primary.ID() {
				continue
			}
			o, e := b.ResolvePortOwner(port)
			if e == nil && o.PID != 0 {
				o.Origin = vmsys.OriginCrossTarget
				owner = o
				break
			}
		}
	}

	// Step 3: classify and act.
	switch {
	case owner.PID == 0:
		// Port is free now (race condition — disappeared between Poll and Force).
		return Result{FinalState: Released, Killed: false}, nil

	case owner.Origin == vmsys.OriginCrossTarget:
		// Cross-target: report only, never kill foreign processes.
		return Result{FinalState: CrossTargetReport, Owner: owner}, nil

	case owner.Origin == vmsys.OriginUnknown:
		return Result{FinalState: UnknownBlocker, Owner: owner}, nil

	default:
		// Same target: force kill.
		if killErr := primary.KillPID(owner.PID); killErr != nil {
			return Result{FinalState: UnknownBlocker, Owner: owner},
				fmt.Errorf("force kill PID %d: %w", owner.PID, killErr)
		}
		return Result{FinalState: Released, Killed: true, Owner: owner}, nil
	}
}
