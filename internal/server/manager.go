//ff:what 서버 런타임 레지스트리 (sync.Map[ServerID → runtimeServer])
//ff:why 다중 프로젝트·서버 동시 실행 시 상태 격리 보장
package server

import (
	"sync"
	"time"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// runtimeServer holds the live state of a running server.
type runtimeServer struct {
	mu      sync.Mutex
	state   State
	handle  vmsys.Handle
	logFile string // abs path to current log file

	// Crash auto-restart bookkeeping (see start.go registerRestart).
	restartCount       int
	restartWindowStart time.Time
}

// Manager owns all live server states.
type Manager struct {
	mu      sync.RWMutex
	servers map[string]*runtimeServer // key: serverID
	backends map[string]vmsys.Backend  // key: execution_target id
}

// NewManager creates an empty Manager.
func NewManager() *Manager {
	return &Manager{
		servers:  make(map[string]*runtimeServer),
		backends: make(map[string]vmsys.Backend),
	}
}

// RegisterBackend stores a backend under its ID for later lookup.
func (m *Manager) RegisterBackend(b vmsys.Backend) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backends[b.ID()] = b
}

// Backend returns the registered backend for target, or nil.
func (m *Manager) Backend(target string) vmsys.Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.backends[target]
}

// AllBackends returns all registered backends.
func (m *Manager) AllBackends() []vmsys.Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]vmsys.Backend, 0, len(m.backends))
	for _, b := range m.backends {
		out = append(out, b)
	}
	return out
}

// GetState returns the current state of a server (StateStopped if unknown).
func (m *Manager) GetState(serverID string) State {
	m.mu.RLock()
	rs := m.servers[serverID]
	m.mu.RUnlock()
	if rs == nil {
		return StateStopped
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.state
}

// setState updates the state of a server, creating the runtime entry if needed.
func (m *Manager) setState(serverID string, state State) {
	m.mu.Lock()
	rs, ok := m.servers[serverID]
	if !ok {
		rs = &runtimeServer{}
		m.servers[serverID] = rs
	}
	m.mu.Unlock()

	rs.mu.Lock()
	rs.state = state
	rs.mu.Unlock()
}

// setHandle stores the spawned process handle.
func (m *Manager) setHandle(serverID string, h vmsys.Handle) {
	m.mu.Lock()
	rs, ok := m.servers[serverID]
	if !ok {
		rs = &runtimeServer{}
		m.servers[serverID] = rs
	}
	m.mu.Unlock()

	rs.mu.Lock()
	rs.handle = h
	rs.mu.Unlock()
}

// clearHandle drops the stored process handle (after a confirmed stop) so a
// stale PID is never reused by a later Stop.
func (m *Manager) clearHandle(serverID string) {
	m.mu.RLock()
	rs := m.servers[serverID]
	m.mu.RUnlock()
	if rs == nil {
		return
	}
	rs.mu.Lock()
	rs.handle = vmsys.Handle{}
	rs.mu.Unlock()
}

// LogPath returns the absolute path of the current log file for a server, or "".
func (m *Manager) LogPath(serverID string) string {
	m.mu.RLock()
	rs := m.servers[serverID]
	m.mu.RUnlock()
	if rs == nil {
		return ""
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.logFile
}

// HandlePID returns the PID of the process VPM spawned for this server, and
// whether VPM actually owns a handle (false for externally-detected servers).
func (m *Manager) HandlePID(serverID string) (int, bool) {
	h, ok := m.getHandle(serverID)
	return h.PID, ok
}

// SyncState sets the runtime state without emitting events.
// Used by app startup to reflect externally-detected process states.
func (m *Manager) SyncState(serverID string, state State) {
	m.setState(serverID, state)
}

// getHandle returns the stored handle for a server.
func (m *Manager) getHandle(serverID string) (vmsys.Handle, bool) {
	m.mu.RLock()
	rs := m.servers[serverID]
	m.mu.RUnlock()
	if rs == nil {
		return vmsys.Handle{}, false
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.handle, rs.handle.PID != 0
}
