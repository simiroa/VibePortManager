package portkiller_test

import (
	"context"
	"net"
	"testing"

	"github.com/user/vpm/internal/portkiller"
	vmsys "github.com/user/vpm/pkg/syscall"
)

// TestRun_ForcePhase: port stays occupied through all 3 polls → Force executes.
// This test takes ~1.5s (3 × 500ms poll interval).
func TestRun_ForcePhase(t *testing.T) {
	// Open a real listener so net.Listen in probePort fails.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener")
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	m := &vmsys.MockExecutor{IDVal: "win"}
	// Backend says port is owned by PID 55 (same target).
	m.OwnerByPort = map[int]vmsys.PortOwner{
		port: {PID: 55, Origin: vmsys.OriginSameTarget},
	}

	handle := vmsys.Handle{PID: 55}
	result, err := portkiller.Run(context.Background(), m, []vmsys.Backend{m}, handle, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Force killed the same-target process → Released.
	if result.FinalState != portkiller.Released {
		t.Errorf("want Released, got %v", result.FinalState)
	}
	if !result.Killed {
		t.Error("expected Killed=true")
	}
}
