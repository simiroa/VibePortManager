//go:build linux

//ff:what Linux-native Executor stub (future)
//ff:why MVP 스코프 밖. 인터페이스만 충족
package syscall

import (
	"context"
	"fmt"
)

type linuxBackend struct{}

func newLinuxBackend() Backend { return &linuxBackend{} }

func (l *linuxBackend) ID() string      { return "linux-native" }
func (l *linuxBackend) Healthy() error  { return nil }
func (l *linuxBackend) KillPID(pid int) error { return fmt.Errorf("linux-native: not implemented") }
func (l *linuxBackend) SignalTree(handle Handle, sig Signal) error {
	return fmt.Errorf("linux-native: not implemented")
}
func (l *linuxBackend) ResolvePortOwner(port int) (PortOwner, error) {
	return PortOwner{}, fmt.Errorf("linux-native: not implemented")
}
func (l *linuxBackend) Spawn(ctx context.Context, spec SpawnSpec) (Handle, error) {
	return Handle{}, fmt.Errorf("linux-native: not implemented")
}
func (l *linuxBackend) ScanListenPorts() ([]ListenEntry, error) {
	return nil, fmt.Errorf("linux-native: not implemented")
}
func (l *linuxBackend) ResolveTreePort(_ int) (int, error) { return 0, nil }
