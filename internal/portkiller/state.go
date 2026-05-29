//ff:what port_killer.mmd 의 상태 enum
//ff:why 다이어그램과 코드 1:1 일치 → validate-specs.go 검증
package portkiller

// State mirrors every state in specs/states/port_killer.mmd exactly.
type State int

const (
	GracefulSent    State = iota // [*] → GracefulSent
	Polling                      // GracefulSent → Polling
	Released                     // Polling → Released  |  ForceKill → Released
	ResolveBlocker               // Polling → ResolveBlocker (3 polls exhausted)
	ForceKill                    // ResolveBlocker → ForceKill (same-target)
	CrossTargetReport            // ResolveBlocker → CrossTargetReport
	UnknownBlocker               // ResolveBlocker → UnknownBlocker  |  ForceKill → UnknownBlocker
)

func (s State) String() string {
	switch s {
	case GracefulSent:
		return "GracefulSent"
	case Polling:
		return "Polling"
	case Released:
		return "Released"
	case ResolveBlocker:
		return "ResolveBlocker"
	case ForceKill:
		return "ForceKill"
	case CrossTargetReport:
		return "CrossTargetReport"
	case UnknownBlocker:
		return "UnknownBlocker"
	default:
		return "Unknown"
	}
}
