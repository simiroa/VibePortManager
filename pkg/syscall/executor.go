//ff:what OS 명령 실행/프로세스 정보의 단일 진입점
//ff:why os/exec 직접 호출을 막아 테스트에서 mock 가능하게
package syscall

import (
	"context"
	"io"
)

// Signal represents a cross-platform process signal.
type Signal int

const (
	SignalTerm Signal = iota
	SignalKill
)

// SpawnSpec assembles the argv and environment for a child process.
type SpawnSpec struct {
	Cmdline   []string          // argv already assembled by shell wrappers
	Cwd       string
	Env       map[string]string // merged final environment
	Stdout    io.Writer
	Stderr    io.Writer
	NewPGroup bool // setpgid on Linux/WSL
}

// Handle identifies a running child process.
type Handle struct {
	PID       int
	PGID      int     // Linux/WSL only; 0 on Windows
	JobObject uintptr // Windows Job Object handle; 0 if unused
}

// Origin describes which execution target owns a port occupant.
type Origin int

const (
	OriginSameTarget  Origin = iota // owned by the same backend that is querying
	OriginCrossTarget               // owned by a different backend (e.g. Windows process seen from WSL)
	OriginUnknown                   // could not be attributed
)

// PortOwner describes the process that holds a port.
type PortOwner struct {
	PID         int
	Description string // e.g. "node.exe (PID 12345)"
	Origin      Origin
}

// ListenEntry is one row from a bulk port scan.
type ListenEntry struct {
	Port        int
	PID         int
	Description string
}

// Executor is the single abstraction for all OS process/port operations.
// All implementations must be thread-safe.
type Executor interface {
	// Spawn starts a child process. Returns immediately; ctx cancellation triggers
	// SignalTree then KillPID in the caller.
	Spawn(ctx context.Context, spec SpawnSpec) (Handle, error)

	// SignalTree sends sig to the process tree / process group (Phase 1 graceful).
	SignalTree(handle Handle, sig Signal) error

	// KillPID force-kills a single PID (Phase 3).
	KillPID(pid int) error

	// ResolvePortOwner finds the PID listening on port. Returns zero-value PortOwner
	// with PID==0 if no listener is found.
	ResolvePortOwner(port int) (PortOwner, error)
}
