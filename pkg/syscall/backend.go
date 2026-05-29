//ff:what Execution Target 별 Executor 구현 선택
//ff:why 1 Project = 1 Backend. Project.execution_target으로 라우팅
package syscall

import (
	"fmt"
	"strings"
)

// Backend extends Executor with identity, health, and bulk scan.
type Backend interface {
	Executor
	// ID returns the execution target identifier, e.g. "windows-native" or "wsl:Ubuntu".
	ID() string
	// Healthy checks that the backend is operational (e.g. WSL distro is running).
	Healthy() error
	// ScanListenPorts returns all TCP LISTEN entries in one bulk call.
	ScanListenPorts() ([]ListenEntry, error)
	// ResolveTreePort returns a TCP LISTEN port owned by the process tree rooted
	// at rootPID (the process or any descendant), or 0 if none. Used to re-detect
	// a VPM-spawned server whose actual port drifted from the configured one.
	ResolveTreePort(rootPID int) (int, error)
}

// NewBackend constructs the appropriate Backend for the given execution target.
// target: "windows-native" | "wsl" | "linux-native"
// wslDistro: only used when target == "wsl"
func NewBackend(target string, wslDistro string) (Backend, error) {
	switch {
	case target == "windows-native":
		return newWinBackend(), nil
	case target == "wsl" || strings.HasPrefix(target, "wsl:"):
		distro := wslDistro
		if distro == "" {
			// try to extract from "wsl:Ubuntu" form
			if idx := strings.Index(target, ":"); idx >= 0 {
				distro = target[idx+1:]
			}
		}
		if distro == "" {
			return nil, fmt.Errorf("wsl backend requires a distro name")
		}
		return newWSLBackend(distro), nil
	case target == "linux-native":
		return newLinuxBackend(), nil
	default:
		return nil, fmt.Errorf("unknown execution target: %q", target)
	}
}
