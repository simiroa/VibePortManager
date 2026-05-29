package portkiller_test

import (
	"errors"
	"testing"

	"github.com/user/vpm/internal/portkiller"
	vmsys "github.com/user/vpm/pkg/syscall"
)

func TestForce_SameTarget_Kills(t *testing.T) {
	m := &vmsys.MockExecutor{IDVal: "win"}
	m.OwnerByPort = map[int]vmsys.PortOwner{
		3001: {PID: 55, Description: "node", Origin: vmsys.OriginSameTarget},
	}

	result, err := portkiller.Force(m, []vmsys.Backend{m}, 3001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FinalState != portkiller.Released {
		t.Errorf("want Released after kill, got %v", result.FinalState)
	}
	if !result.Killed {
		t.Error("Killed should be true")
	}
	if len(m.KillPIDs) == 0 || m.KillPIDs[0] != 55 {
		t.Errorf("expected KillPID(55), got %v", m.KillPIDs)
	}
}

func TestForce_UnknownBlocker(t *testing.T) {
	m := &vmsys.MockExecutor{IDVal: "win"}
	m.OwnerByPort = map[int]vmsys.PortOwner{
		3002: {PID: 77, Description: "mystery", Origin: vmsys.OriginUnknown},
	}

	result, err := portkiller.Force(m, []vmsys.Backend{m}, 3002)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FinalState != portkiller.UnknownBlocker {
		t.Errorf("want UnknownBlocker, got %v", result.FinalState)
	}
	if len(m.KillPIDs) != 0 {
		t.Error("must not kill unknown-origin process")
	}
}

func TestForce_KillError(t *testing.T) {
	m := &vmsys.MockExecutor{IDVal: "win"}
	m.OwnerByPort = map[int]vmsys.PortOwner{
		3003: {PID: 88, Origin: vmsys.OriginSameTarget},
	}
	m.KillErr = errors.New("access denied")

	_, err := portkiller.Force(m, []vmsys.Backend{m}, 3003)
	if err == nil {
		t.Error("expected error from KillPID failure")
	}
}

func TestForce_PrimaryResolveError(t *testing.T) {
	m := &vmsys.MockExecutor{IDVal: "win"}
	m.OwnerErr = errors.New("netstat failed")

	_, err := portkiller.Force(m, []vmsys.Backend{m}, 3004)
	if err == nil {
		t.Error("expected error when ResolvePortOwner fails")
	}
}

func TestForce_SecondaryResolveError_Ignored(t *testing.T) {
	// Primary returns no owner; secondary errors — should not propagate.
	primary := &vmsys.MockExecutor{IDVal: "win"}
	primary.OwnerByPort = map[int]vmsys.PortOwner{}

	other := &vmsys.MockExecutor{IDVal: "wsl"}
	other.OwnerErr = errors.New("wsl down")

	result, err := portkiller.Force(primary, []vmsys.Backend{primary, other}, 3005)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No owner found on any backend → Released.
	if result.FinalState != portkiller.Released {
		t.Errorf("want Released, got %v", result.FinalState)
	}
}
