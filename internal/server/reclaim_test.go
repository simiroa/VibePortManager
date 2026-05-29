package server

import (
	"sync"
	"testing"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// killableBackend reports a port owner until its tree is killed, then reports
// the port as free — exercising reclaimPort's kill→recheck loop.
type killableBackend struct {
	*vmsys.MockExecutor
	mu     sync.Mutex
	killed bool
	kills  int
}

func (k *killableBackend) ResolvePortOwner(int) (vmsys.PortOwner, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.killed {
		return vmsys.PortOwner{}, nil
	}
	return vmsys.PortOwner{PID: 100, Description: "fake-listener", Origin: vmsys.OriginSameTarget}, nil
}

func (k *killableBackend) SignalTree(vmsys.Handle, vmsys.Signal) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.killed = true
	k.kills++
	return nil
}

func TestReclaimPortKillsThenFrees(t *testing.T) {
	m := NewManager()
	kb := &killableBackend{MockExecutor: &vmsys.MockExecutor{}}

	// Port 59997 is not actually listening, so the post-kill IsPortOccupied probe
	// confirms it is free.
	if !m.reclaimPort(kb, 59997) {
		t.Fatalf("expected reclaimPort to free the port")
	}
	if kb.kills != 1 {
		t.Errorf("expected exactly 1 tree-kill, got %d", kb.kills)
	}
}
