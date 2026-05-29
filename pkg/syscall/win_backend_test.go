//go:build windows

package syscall

import (
	"net"
	"os"
	"testing"
)

// TestResolveTreePortFindsOwnListener binds a socket in this test process, then
// asks the backend for a LISTEN port owned by this process's tree (rooted at our
// own PID). The mechanism (process snapshot ∩ netstat) must find a port.
func TestResolveTreePortFindsOwnListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	want := ln.Addr().(*net.TCPAddr).Port

	got, err := newWinBackend().ResolveTreePort(os.Getpid())
	if err != nil {
		t.Fatalf("ResolveTreePort: %v", err)
	}
	if got == 0 {
		t.Fatalf("expected a tree-owned LISTEN port, got 0 (test is listening on %d)", want)
	}
	t.Logf("ResolveTreePort(self) = %d (this test listens on %d)", got, want)
}

func TestResolveTreePortNoProcess(t *testing.T) {
	// PID 0 / negative → no tree, no port, no error.
	if p, err := newWinBackend().ResolveTreePort(0); err != nil || p != 0 {
		t.Errorf("ResolveTreePort(0) = (%d,%v), want (0,nil)", p, err)
	}
}
