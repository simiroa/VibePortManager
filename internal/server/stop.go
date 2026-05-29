//ff:what 서버 중지: Triple-Pass Port Killer 실행
//ff:why graceful → poll(3×500ms) → force kill 순서로 포트 회수 보장
package server

import (
	"context"
	"fmt"
	"time"

	"github.com/user/vpm/internal/config"
	"github.com/user/vpm/internal/portkiller"
	vmsys "github.com/user/vpm/pkg/syscall"
)

// Stop gracefully stops the server via Triple-Pass Port Killer.
//
// Works for servers VPM started (we hold the process handle) AND for servers
// started outside VPM that we detected via the port probe (no handle). In the
// no-handle case we still kill whoever is listening on the port — otherwise
// Stop would be a no-op and a following Start would hit PORT_COLLISION.
func (m *Manager) Stop(ctx context.Context, proj config.Project, srv config.Server, onState func(StateEvent)) error {
	cur := m.GetState(srv.ID)
	if cur == StateStopped {
		return nil
	}
	if cur == StateStarting || cur == StateStopping {
		return fmt.Errorf("server %s is busy (%s)", srv.ID, cur)
	}

	primary := m.Backend(EffectiveTarget(proj))
	if primary == nil {
		m.setState(srv.ID, StateError)
		onState(StateEvent{ServerID: srv.ID, State: StateError, Error: "no backend"})
		return fmt.Errorf("no backend for %s", proj.ExecutionTarget)
	}

	// handle.PID == 0 when VPM did not spawn this server. The Triple-Pass Force
	// phase resolves the real port owner, so a zero handle is fine — but if the
	// port is already free, there is nothing to do.
	handle, hasHandle := m.getHandle(srv.ID)
	if !hasHandle {
		if occupied, _ := IsPortOccupied(primary, srv.Port); !occupied {
			m.setState(srv.ID, StateStopped)
			onState(StateEvent{ServerID: srv.ID, State: StateStopped})
			return nil
		}
	}

	m.setState(srv.ID, StateStopping)
	onState(StateEvent{ServerID: srv.ID, State: StateStopping})

	// Phase A — graceful + first force pass on the handle we own (if any).
	_, _ = portkiller.Run(ctx, primary, m.AllBackends(), handle, srv.Port)

	// Phase B — exhaustive reclaim. The owner of the LISTEN socket is often a
	// *different* PID than the process we spawned (npm→cmd→node trees) and there
	// may be several stale instances. Tree-kill every remaining listener, looping
	// until the port is actually free.
	if !m.reclaimPort(primary, srv.Port) {
		owner, _ := primary.ResolvePortOwner(srv.Port)
		msg := fmt.Sprintf("could not free port %d", srv.Port)
		if owner.PID != 0 {
			msg = fmt.Sprintf("port %d still held by %s (may be a cross-target/elevated process)", srv.Port, owner.Description)
		}
		m.setState(srv.ID, StateError)
		onState(StateEvent{ServerID: srv.ID, State: StateError, Error: msg})
		return fmt.Errorf("%s", msg)
	}

	m.clearHandle(srv.ID)
	m.setState(srv.ID, StateStopped)
	onState(StateEvent{ServerID: srv.ID, State: StateStopped})
	return nil
}

// reclaimPort tree-kills every process listening on port until it is free.
// Each iteration resolves the current owner and kills its whole process tree
// (taskkill /T /F), which clears npm→cmd→node trees and successive stale
// instances. Returns true once nothing is listening, false if it gives up.
func (m *Manager) reclaimPort(primary vmsys.Backend, port int) bool {
	const maxPasses = 6
	for i := 0; i < maxPasses; i++ {
		owner, err := primary.ResolvePortOwner(port)
		if err == nil && owner.PID == 0 {
			if occupied, _ := IsPortOccupied(primary, port); !occupied {
				return true // port is free
			}
			// Occupied but owner unresolved (rare) — nothing we can target.
			return false
		}
		if owner.PID != 0 {
			// /T /F: kill the whole tree, not just the resolved PID.
			_ = primary.SignalTree(vmsys.Handle{PID: owner.PID}, vmsys.SignalKill)
		}
		time.Sleep(350 * time.Millisecond)
	}
	occupied, _ := IsPortOccupied(primary, port)
	return !occupied
}
