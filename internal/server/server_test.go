package server

import (
	"net"
	"testing"

	"github.com/user/vpm/internal/config"
	vmsys "github.com/user/vpm/pkg/syscall"
)

// ── SyncState ─────────────────────────────────────────────────────────────────

func TestSyncState_SetsStateWithoutEvent(t *testing.T) {
	m := NewManager()
	const id = "srv-1"

	// Before SyncState, state is STOPPED (default).
	if got := m.GetState(id); got != StateStopped {
		t.Fatalf("initial state = %v, want STOPPED", got)
	}

	m.SyncState(id, StateRunning)
	if got := m.GetState(id); got != StateRunning {
		t.Fatalf("after SyncState(RUNNING) = %v, want RUNNING", got)
	}
}

func TestSyncState_CanTransitionBack(t *testing.T) {
	m := NewManager()
	const id = "srv-2"

	m.SyncState(id, StateRunning)
	m.SyncState(id, StateStopped)

	if got := m.GetState(id); got != StateStopped {
		t.Fatalf("after SyncState(STOPPED) = %v, want STOPPED", got)
	}
}

func TestSyncState_IndependentServers(t *testing.T) {
	m := NewManager()

	m.SyncState("a", StateRunning)
	m.SyncState("b", StateError)

	if m.GetState("a") != StateRunning {
		t.Error("server a should be RUNNING")
	}
	if m.GetState("b") != StateError {
		t.Error("server b should be ERROR")
	}
}

// ── IsPortOccupied ────────────────────────────────────────────────────────────

func TestIsPortOccupied_FreePort(t *testing.T) {
	mock := &vmsys.MockExecutor{}
	// Find a free OS port to use as test subject.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener:", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // release it — IsPortOccupied will find it free

	occupied, _ := IsPortOccupied(mock, port)
	if occupied {
		t.Errorf("port %d should be free, got occupied=true", port)
	}
	// Mock.ResolvePortOwner should NOT be called when the port is free.
	if len(mock.OwnerPorts) != 0 {
		t.Errorf("expected no ResolvePortOwner calls, got %d", len(mock.OwnerPorts))
	}
}

func TestIsPortOccupied_OccupiedPort(t *testing.T) {
	// Hold a port open so IsPortOccupied sees it as occupied.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener:", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	mock := &vmsys.MockExecutor{
		OwnerByPort: map[int]vmsys.PortOwner{
			port: {PID: 1234, Description: "test-proc"},
		},
	}

	occupied, owner := IsPortOccupied(mock, port)
	if !occupied {
		t.Errorf("port %d should be occupied", port)
	}
	if owner.PID != 1234 {
		t.Errorf("owner.PID = %d, want 1234", owner.PID)
	}
}

func TestIsPortOccupied_OccupiedPortOwnerNotFound(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener:", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	// Mock returns no owner (PID=0).
	mock := &vmsys.MockExecutor{}

	occupied, owner := IsPortOccupied(mock, port)
	if !occupied {
		t.Errorf("port %d should still be occupied even if owner unknown", port)
	}
	if owner.Origin != vmsys.OriginUnknown {
		t.Errorf("owner.Origin = %v, want OriginUnknown", owner.Origin)
	}
}

// ── EffectiveTarget ───────────────────────────────────────────────────────────

func TestEffectiveTarget_WindowsNative(t *testing.T) {
	p := config.Project{ExecutionTarget: "windows-native"}
	if got := EffectiveTarget(p); got != "windows-native" {
		t.Errorf("EffectiveTarget = %q, want %q", got, "windows-native")
	}
}

func TestEffectiveTarget_WSLWithDistro(t *testing.T) {
	p := config.Project{ExecutionTarget: "wsl", WSLDistro: "Ubuntu"}
	if got := EffectiveTarget(p); got != "wsl:Ubuntu" {
		t.Errorf("EffectiveTarget = %q, want %q", got, "wsl:Ubuntu")
	}
}

func TestEffectiveTarget_WSLWithoutDistro(t *testing.T) {
	// wsl with empty distro → falls through to plain ExecutionTarget
	p := config.Project{ExecutionTarget: "wsl", WSLDistro: ""}
	if got := EffectiveTarget(p); got != "wsl" {
		t.Errorf("EffectiveTarget = %q, want %q", got, "wsl")
	}
}

func TestEffectiveTarget_Empty(t *testing.T) {
	p := config.Project{}
	if got := EffectiveTarget(p); got != "" {
		t.Errorf("EffectiveTarget = %q, want empty", got)
	}
}
