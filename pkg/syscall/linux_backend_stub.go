//go:build !linux

//ff:what Linux backend stub for non-Linux platforms
//ff:why 빌드 태그 없는 플랫폼에서 컴파일 오류 방지
package syscall

import (
	"context"
	"fmt"
)

func newLinuxBackend() Backend { return &linuxStub{} }

type linuxStub struct{}

func (l *linuxStub) ID() string    { return "linux-native" }
func (l *linuxStub) Healthy() error { return fmt.Errorf("linux-native: unsupported platform") }
func (l *linuxStub) KillPID(_ int) error { return fmt.Errorf("linux-native: unsupported") }
func (l *linuxStub) SignalTree(_ Handle, _ Signal) error {
	return fmt.Errorf("linux-native: unsupported")
}
func (l *linuxStub) ResolvePortOwner(_ int) (PortOwner, error) {
	return PortOwner{}, fmt.Errorf("linux-native: unsupported")
}
func (l *linuxStub) Spawn(_ context.Context, _ SpawnSpec) (Handle, error) {
	return Handle{}, fmt.Errorf("linux-native: unsupported")
}
func (l *linuxStub) ScanListenPorts() ([]ListenEntry, error) {
	return nil, fmt.Errorf("linux-native: unsupported")
}
func (l *linuxStub) ResolveTreePort(_ int) (int, error) {
	return 0, fmt.Errorf("linux-native: unsupported")
}
func (l *linuxStub) ResolveProcessCommand(_ int) (string, error) {
	return "", fmt.Errorf("linux-native: unsupported")
}
