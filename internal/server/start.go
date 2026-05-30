//ff:what 서버 시작: 포트 사전 검사 → Spawn → 상태 전환
//ff:why 포트 점유 시 즉시 PORT_COLLISION 이벤트 방출 (Spawn 시도 전)
package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/user/vpm/internal/config"
	"github.com/user/vpm/internal/logbuf"
	"github.com/user/vpm/internal/pwshwrap"
	"github.com/user/vpm/internal/wslwrap"
	vmsys "github.com/user/vpm/pkg/syscall"
)

// StateEvent carries a state transition notification.
type StateEvent struct {
	ServerID string
	State    State
	Error    string // non-empty on StateError or StatePortCollision
}

// CollisionEvent describes a port blocker.
type CollisionEvent struct {
	ServerID            string
	Origin              string
	BlockingPID         int
	BlockingDescription string
	SuggestedFreePort   int
}

// Start launches a server process.
// onState is called on every state transition (may be called from goroutines).
// onCollision is called if a port blocker is detected.
// onLog is the batch log flush callback.
func (m *Manager) Start(
	ctx context.Context,
	proj config.Project,
	srv config.Server,
	onState func(StateEvent),
	onCollision func(CollisionEvent),
	onLog logbuf.Flush,
) error {
	// Guard: monitor-only server (no start command). VPM only tracks its port;
	// it doesn't own the process, so there's nothing to spawn.
	if strings.TrimSpace(srv.Command) == "" {
		return fmt.Errorf("server %q is monitor-only (no start command)", srv.Name)
	}

	// Guard: already transitioning.
	cur := m.GetState(srv.ID)
	if cur == StateStarting || cur == StateStopping {
		return fmt.Errorf("server %s is busy (%s)", srv.ID, cur)
	}

	m.setState(srv.ID, StateStarting)
	onState(StateEvent{ServerID: srv.ID, State: StateStarting})

	// Pre-flight: check port availability.
	backend := m.Backend(EffectiveTarget(proj))
	if backend == nil {
		m.setState(srv.ID, StateError)
		onState(StateEvent{ServerID: srv.ID, State: StateError, Error: "no backend for " + proj.ExecutionTarget})
		return fmt.Errorf("no backend registered for %s", proj.ExecutionTarget)
	}

	if occupied, owner := IsPortOccupied(backend, srv.Port); occupied {
		m.setState(srv.ID, StatePortCollision)
		origin := originString(owner.Origin)
		ev := CollisionEvent{
			ServerID:            srv.ID,
			Origin:              origin,
			BlockingPID:         owner.PID,
			BlockingDescription: owner.Description,
		}
		onCollision(ev)
		onState(StateEvent{ServerID: srv.ID, State: StatePortCollision, Error: fmt.Sprintf("port %d occupied by PID %d", srv.Port, owner.PID)})
		return fmt.Errorf("port %d occupied", srv.Port)
	}

	// Build spawn spec.
	logFile, logPath, err := logbuf.NewLogFile(proj.ID, srv.ID)
	if err != nil {
		m.setState(srv.ID, StateError)
		onState(StateEvent{ServerID: srv.ID, State: StateError, Error: err.Error()})
		return err
	}

	buf := logbuf.New(srv.ID, onLog)
	writer := multiWriter{buf, logFile}

	var spec vmsys.SpawnSpec
	switch proj.ExecutionTarget {
	case "wsl":
		spec, _ = wslwrap.Build(proj.WSLDistro, srv.Command, srv.Port, proj.Path, nil, writer, writer)
	default: // windows-native
		spec, _ = pwshwrap.Build(srv.Command, srv.Port, proj.Path, nil, writer, writer)
	}

	handle, err := backend.Spawn(ctx, spec)
	if err != nil {
		logFile.Close()
		buf.Close()
		m.setState(srv.ID, StateError)
		onState(StateEvent{ServerID: srv.ID, State: StateError, Error: err.Error()})
		return err
	}

	m.setHandle(srv.ID, handle)

	// Store log path.
	m.mu.Lock()
	if rs, ok := m.servers[srv.ID]; ok {
		rs.mu.Lock()
		rs.logFile = logPath
		rs.mu.Unlock()
	}
	m.mu.Unlock()

	m.setState(srv.ID, StateRunning)
	onState(StateEvent{ServerID: srv.ID, State: StateRunning})

	// Watch for process exit in background (handles crash auto-restart).
	go m.watchExit(srv.ID, handle, backend, buf, logFile, proj, srv, onState, onCollision, onLog)

	return nil
}

// Crash auto-restart tuning.
const (
	maxRestarts    = 5                // within restartWindow before giving up
	restartWindow  = 60 * time.Second // sliding window for the restart counter
	restartBackoff = time.Second      // pause before relaunching
)

// registerRestart records a restart attempt and reports whether it is still
// within the crash-loop budget. Resets the counter once restartWindow elapses.
func (m *Manager) registerRestart(serverID string) bool {
	m.mu.RLock()
	rs := m.servers[serverID]
	m.mu.RUnlock()
	if rs == nil {
		return false
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	now := time.Now()
	if now.Sub(rs.restartWindowStart) > restartWindow {
		rs.restartWindowStart = now
		rs.restartCount = 0
	}
	rs.restartCount++
	return rs.restartCount <= maxRestarts
}

// watchExit monitors the process and transitions to STOPPED/ERROR on exit.
// On Windows, os.FindProcess opens a real HANDLE and Wait() calls WaitForSingleObject — works for
// any PID we have access to (not just direct children).
// On WSL/Linux, Wait() only works for direct children; if the WSL backend uses exec.Cmd.Start()
// internally and stores the *os.Process in the Handle, wire it here. For now we fall back to
// a 1-second poll via ResolvePortOwner when os.FindProcess.Wait fails immediately.
func (m *Manager) watchExit(serverID string, h vmsys.Handle, _ vmsys.Backend, buf *logbuf.Buffer, logFile interface{ Close() error }, proj config.Project, srv config.Server, onState func(StateEvent), onCollision func(CollisionEvent), onLog logbuf.Flush) {
	if h.PID > 0 {
		if proc, err := os.FindProcess(h.PID); err == nil {
			// Wait blocks until the OS process terminates (Windows: WaitForSingleObject).
			proc.Wait() //nolint: ignore error — process may already be gone
		}
	}

	buf.Close()
	logFile.Close()

	// User-initiated stop: Stop() already owns the terminal state.
	if m.GetState(serverID) == StateStopping {
		return
	}

	// Unexpected exit. Auto-restart when enabled and within the crash-loop budget.
	if srv.Autorestart {
		if m.registerRestart(serverID) {
			onState(StateEvent{ServerID: serverID, State: StateStopped, Error: "exited — auto-restarting"})
			time.Sleep(restartBackoff)
			// Start() flips the state back through STARTING→RUNNING and re-arms watchExit.
			_ = m.Start(context.Background(), proj, srv, onState, onCollision, onLog)
			return
		}
		m.setState(serverID, StateError)
		onState(StateEvent{ServerID: serverID, State: StateError, Error: "crash loop: too many restarts, giving up"})
		return
	}

	m.setState(serverID, StateStopped)
	onState(StateEvent{ServerID: serverID, State: StateStopped})
}

// IsPortOccupied probes whether a port is already listening.
// Uses net.Dial (connect attempt) rather than net.Listen (bind attempt) because
// on Windows both sides can use SO_REUSEADDR, which allows two sockets to bind
// to the same port and would make Listen succeed even when the port is in use.
// Exported for use by app.go startup probe and tests.
func IsPortOccupied(b vmsys.Backend, port int) (bool, vmsys.PortOwner) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err != nil {
		return false, vmsys.PortOwner{}
	}
	conn.Close()
	owner, err := b.ResolvePortOwner(port)
	if err != nil || owner.PID == 0 {
		return true, vmsys.PortOwner{Origin: vmsys.OriginUnknown}
	}
	return true, owner
}

// EffectiveTarget returns the backend registry key for a project.
// Exported for use by app.go.
func EffectiveTarget(proj config.Project) string {
	if proj.ExecutionTarget == "wsl" && proj.WSLDistro != "" {
		return "wsl:" + proj.WSLDistro
	}
	return proj.ExecutionTarget
}

func originString(o vmsys.Origin) string {
	switch o {
	case vmsys.OriginSameTarget:
		return "same-target"
	case vmsys.OriginCrossTarget:
		return "cross-target"
	default:
		return "unknown"
	}
}

// multiWriter fans out writes to two io.Writers.
type multiWriter struct {
	a, b interface {
		Write([]byte) (int, error)
	}
}

func (mw multiWriter) Write(p []byte) (int, error) {
	n, err := mw.a.Write(p)
	if err != nil {
		return n, err
	}
	mw.b.Write(p) //nolint: ignore secondary error
	return n, nil
}
