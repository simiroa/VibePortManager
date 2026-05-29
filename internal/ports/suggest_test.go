package ports

import (
	"net"
	"testing"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// TestSuggestFreeSkipsOccupied binds a real port so isFree's net.Listen probe
// fails on it, and backs that with a mock owner so the fallback also reports it
// occupied. SuggestFree must skip past it to the next free port.
func TestSuggestFreeSkipsOccupied(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	occupied := ln.Addr().(*net.TCPAddr).Port
	if occupied >= 65500 {
		t.Skipf("OS assigned a port too near the ceiling: %d", occupied)
	}

	b := &vmsys.MockExecutor{OwnerByPort: map[int]vmsys.PortOwner{occupied: {PID: 999}}}

	got, err := SuggestFree(b, occupied-1)
	if err != nil {
		t.Fatalf("SuggestFree: %v", err)
	}
	if got == occupied {
		t.Errorf("returned the occupied port %d", occupied)
	}
	if got <= occupied-1 {
		t.Errorf("got %d, expected a port greater than startFrom %d", got, occupied-1)
	}
}

func TestSuggestFreeHappyPath(t *testing.T) {
	b := &vmsys.MockExecutor{} // mock reports every port free via PID 0
	const start = 28000
	got, err := SuggestFree(b, start)
	if err != nil {
		t.Fatalf("SuggestFree: %v", err)
	}
	if got <= start || got > start+suggestScanRange {
		t.Errorf("got %d, want in (%d, %d]", got, start, start+suggestScanRange)
	}
}
