//ff:what WSL2 네트워크 모드(NAT vs mirrored) 감지
//ff:why mirrored 모드에서는 localhost:PORT가 Windows에서 직접 접근 가능
package wslwrap

import (
	"os"
	"strings"
)

// NetworkMode describes how WSL2 exposes ports to the Windows host.
type NetworkMode int

const (
	NetworkModeNAT      NetworkMode = iota // default: WSL2 uses NAT; host accesses via localhost relay
	NetworkModeMirrored                    // .wslconfig networkingMode=mirrored: shared IP stack
)

// DetectNetworkMode reads %USERPROFILE%\.wslconfig and checks networkingMode.
// If the file is absent or the key is missing, NAT is assumed.
func DetectNetworkMode() NetworkMode {
	home := os.Getenv("USERPROFILE")
	if home == "" {
		return NetworkModeNAT
	}
	data, err := os.ReadFile(home + `\.wslconfig`)
	if err != nil {
		return NetworkModeNAT
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.EqualFold(line, "networkingMode=mirrored") {
			return NetworkModeMirrored
		}
	}
	return NetworkModeNAT
}
