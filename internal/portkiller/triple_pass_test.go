package portkiller_test

import (
	"context"
	"errors"
	"testing"

	"github.com/user/vpm/internal/portkiller"
	vmsys "github.com/user/vpm/pkg/syscall"
)

func TestTriplePass_Released(t *testing.T) {
	m := &vmsys.MockExecutor{}
	// Port owner returns PID 0 (free) on first call.
	m.OwnerByPort = map[int]vmsys.PortOwner{3000: {PID: 0}}

	handle := vmsys.Handle{PID: 42}
	result, err := portkiller.Run(context.Background(), m, []vmsys.Backend{m}, handle, 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FinalState != portkiller.Released {
		t.Errorf("want Released, got %v", result.FinalState)
	}
	if len(m.SignalCalls) == 0 {
		t.Error("expected SignalTerm to be sent")
	}
}

func TestTriplePass_ForceKill(t *testing.T) {
	m := &vmsys.MockExecutor{}
	// Port stays occupied through all polls.
	m.OwnerByPort = map[int]vmsys.PortOwner{3000: {PID: 42, Description: "node", Origin: vmsys.OriginSameTarget}}

	handle := vmsys.Handle{PID: 42}
	result, err := portkiller.Run(context.Background(), m, []vmsys.Backend{m}, handle, 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Either ForceKill or UnknownBlocker depending on KillPID outcome.
	if result.FinalState == portkiller.GracefulSent {
		t.Error("should have progressed past GracefulSent")
	}
}

// TestForce_CrossTarget tests Phase 3 (Force) cross-target detection directly.
// We bypass Run/Poll since the TCP probe has no real listener to wait for.
func TestForce_CrossTarget(t *testing.T) {
	primary := &vmsys.MockExecutor{IDVal: "windows-native"}
	primary.OwnerByPort = map[int]vmsys.PortOwner{} // primary sees no owner

	other := &vmsys.MockExecutor{IDVal: "wsl:Ubuntu"}
	other.OwnerByPort = map[int]vmsys.PortOwner{3000: {PID: 99, Description: "chrome"}}

	result, err := portkiller.Force(primary, []vmsys.Backend{primary, other}, 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FinalState != portkiller.CrossTargetReport {
		t.Errorf("want CrossTargetReport, got %v", result.FinalState)
	}
	if len(other.KillPIDs) != 0 {
		t.Error("must not kill cross-target process")
	}
}

func TestGraceful_SignalError(t *testing.T) {
	m := &vmsys.MockExecutor{}
	m.SignalErr = errors.New("access denied")
	err := portkiller.Graceful(m, vmsys.Handle{PID: 1})
	if err == nil {
		t.Error("expected error from SignalTree")
	}
}
