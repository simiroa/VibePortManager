//go:build !windows

//ff:what Windows backend stub for non-Windows platforms
//ff:why Windows 전용 API 없는 플랫폼에서 컴파일 오류 방지
package syscall

import (
	"context"
	"fmt"
)

func newWinBackend() Backend { return &winStub{} }

type winStub struct{}

func (w *winStub) ID() string    { return "windows-native" }
func (w *winStub) Healthy() error { return fmt.Errorf("windows-native: unsupported platform") }
func (w *winStub) KillPID(_ int) error { return fmt.Errorf("windows-native: unsupported") }
func (w *winStub) SignalTree(_ Handle, _ Signal) error {
	return fmt.Errorf("windows-native: unsupported")
}
func (w *winStub) ResolvePortOwner(_ int) (PortOwner, error) {
	return PortOwner{}, fmt.Errorf("windows-native: unsupported")
}
func (w *winStub) Spawn(_ context.Context, _ SpawnSpec) (Handle, error) {
	return Handle{}, fmt.Errorf("windows-native: unsupported")
}
func (w *winStub) ScanListenPorts() ([]ListenEntry, error) {
	return nil, fmt.Errorf("windows-native: unsupported")
}
func (w *winStub) ResolveTreePort(_ int) (int, error) {
	return 0, fmt.Errorf("windows-native: unsupported")
}
func (w *winStub) ResolveProcessCommand(_ int) (string, error) {
	return "", fmt.Errorf("windows-native: unsupported")
}
