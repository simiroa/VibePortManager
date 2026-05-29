package portkiller

import (
	"context"
	"errors"
	"net"
	"testing"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// TestProbePort_Free: nothing bound → net.Listen succeeds → free=true.
func TestProbePort_Free(t *testing.T) {
	// Find a free port first.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot allocate test port")
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // release it immediately

	m := &vmsys.MockExecutor{}
	free, err := probePort(m, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !free {
		t.Error("expected port to be free")
	}
}

// TestProbePort_Occupied_NonZeroPID: listener bound → net.Listen fails → backend returns PID=42 → free=false.
func TestProbePort_Occupied_NonZeroPID(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener")
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	m := &vmsys.MockExecutor{}
	m.OwnerByPort = map[int]vmsys.PortOwner{port: {PID: 42}}

	free, err := probePort(m, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if free {
		t.Error("port should not be free: listener is active")
	}
}

// TestProbePort_Occupied_ZeroPID: listener bound → net.Listen fails → backend returns PID=0 → free=true.
func TestProbePort_Occupied_ZeroPID(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener")
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	m := &vmsys.MockExecutor{}
	m.OwnerByPort = map[int]vmsys.PortOwner{} // port not in map → PID=0

	free, err := probePort(m, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !free {
		t.Error("backend said PID=0 → should be free")
	}
}

// TestProbePort_BackendError: listener bound → net.Listen fails → backend errors → propagated.
func TestProbePort_BackendError(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener")
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	m := &vmsys.MockExecutor{}
	m.OwnerErr = errors.New("netstat failed")

	_, err = probePort(m, port)
	if err == nil {
		t.Error("expected error from backend")
	}
}

// TestPoll_CtxCanceled: canceled context → Poll exits early with error.
func TestPoll_CtxCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	m := &vmsys.MockExecutor{}
	_, err := Poll(ctx, m, 19999)
	if err == nil {
		t.Error("expected context error")
	}
}

// TestPoll_ProbeError: listener occupied, backend errors → Poll propagates error.
func TestPoll_ProbeError(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("cannot open listener")
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	m := &vmsys.MockExecutor{}
	m.OwnerErr = errors.New("backend error")

	_, err = Poll(context.Background(), m, port)
	if err == nil {
		t.Error("expected error from Poll when backend errors")
	}
}
