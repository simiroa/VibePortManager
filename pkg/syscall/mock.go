//ff:what 테스트용 MockExecutor — 호출 기록 + 사전 주입 응답
//ff:why 단위 테스트에서 실제 OS 호출 없이 Executor 동작 검증
package syscall

import (
	"context"
	"sync"
)

// SpawnCall records a single Spawn invocation.
type SpawnCall struct {
	Spec SpawnSpec
}

// SignalCall records a SignalTree invocation.
type SignalCall struct {
	Handle Handle
	Signal Signal
}

// MockExecutor is a thread-safe fake Backend for use in tests.
type MockExecutor struct {
	mu sync.Mutex

	// IDVal overrides the default "mock" ID for multi-backend tests.
	IDVal string

	// Injected responses (set before test).
	SpawnErr    error
	SpawnHandle Handle

	SignalErr error
	KillErr   error

	OwnerByPort  map[int]PortOwner // port -> owner; zero-value = not found
	OwnerErr     error
	ListenPorts  []ListenEntry // returned by ScanListenPorts
	ListenErr    error
	TreePort     int   // returned by ResolveTreePort
	TreePortErr  error
	CmdByPID     map[int]string // pid -> command line; returned by ResolveProcessCommand
	CmdErr       error

	// Call records (read after test).
	SpawnCalls  []SpawnCall
	SignalCalls []SignalCall
	KillPIDs    []int
	OwnerPorts  []int
}

func (m *MockExecutor) ID() string {
	if m.IDVal != "" {
		return m.IDVal
	}
	return "mock"
}
func (m *MockExecutor) Healthy() error { return nil }

func (m *MockExecutor) Spawn(_ context.Context, spec SpawnSpec) (Handle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SpawnCalls = append(m.SpawnCalls, SpawnCall{Spec: spec})
	return m.SpawnHandle, m.SpawnErr
}

func (m *MockExecutor) SignalTree(handle Handle, sig Signal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SignalCalls = append(m.SignalCalls, SignalCall{Handle: handle, Signal: sig})
	return m.SignalErr
}

func (m *MockExecutor) KillPID(pid int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.KillPIDs = append(m.KillPIDs, pid)
	return m.KillErr
}

func (m *MockExecutor) ResolvePortOwner(port int) (PortOwner, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OwnerPorts = append(m.OwnerPorts, port)
	if m.OwnerErr != nil {
		return PortOwner{}, m.OwnerErr
	}
	if m.OwnerByPort == nil {
		return PortOwner{}, nil
	}
	return m.OwnerByPort[port], nil
}

func (m *MockExecutor) ScanListenPorts() ([]ListenEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ListenPorts, m.ListenErr
}

func (m *MockExecutor) ResolveTreePort(_ int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.TreePort, m.TreePortErr
}

func (m *MockExecutor) ResolveProcessCommand(pid int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CmdErr != nil {
		return "", m.CmdErr
	}
	if m.CmdByPID == nil {
		return "", nil
	}
	return m.CmdByPID[pid], nil
}

// Reset clears all call records (keeps injected responses).
func (m *MockExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SpawnCalls = m.SpawnCalls[:0]
	m.SignalCalls = m.SignalCalls[:0]
	m.KillPIDs = m.KillPIDs[:0]
	m.OwnerPorts = m.OwnerPorts[:0]
}
