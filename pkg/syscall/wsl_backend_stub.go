//go:build !windows

//ff:what WSL backend stub for non-Windows platforms
//ff:why WSL은 Windows 전용 기능
package syscall

import (
	"context"
	"fmt"
)

func newWSLBackend(distro string) Backend { return &wslStub{distro: distro} }

type wslStub struct{ distro string }

func (w *wslStub) ID() string    { return "wsl:" + w.distro }
func (w *wslStub) Healthy() error { return fmt.Errorf("wsl: unsupported platform") }
func (w *wslStub) KillPID(_ int) error { return fmt.Errorf("wsl: unsupported") }
func (w *wslStub) SignalTree(_ Handle, _ Signal) error { return fmt.Errorf("wsl: unsupported") }
func (w *wslStub) ResolvePortOwner(_ int) (PortOwner, error) {
	return PortOwner{}, fmt.Errorf("wsl: unsupported")
}
func (w *wslStub) Spawn(_ context.Context, _ SpawnSpec) (Handle, error) {
	return Handle{}, fmt.Errorf("wsl: unsupported")
}
func (w *wslStub) ScanListenPorts() ([]ListenEntry, error) {
	return nil, fmt.Errorf("wsl: unsupported")
}
func (w *wslStub) ResolveTreePort(_ int) (int, error) {
	return 0, fmt.Errorf("wsl: unsupported")
}
func (w *wslStub) ResolveProcessCommand(_ int) (string, error) {
	return "", fmt.Errorf("wsl: unsupported")
}
