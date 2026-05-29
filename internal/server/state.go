//ff:what 서버 상태 머신 열거값 (specs/states/server.mmd SSOT)
//ff:why UI 버튼 활성화 여부는 상태만으로 결정: STARTING/STOPPING=비활성
package server

// State represents the lifecycle state of a managed server process.
// Values must stay in sync with specs/states/server.mmd.
type State int

const (
	StateStopped       State = iota // no process
	StateStarting                   // Spawn called, not yet confirmed
	StateRunning                    // TCP probe confirmed port open
	StateStopping                   // TriplePass in progress
	StatePortCollision              // port blocked by foreign process
	StateError                      // abnormal exit or spawn failure
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "STOPPED"
	case StateStarting:
		return "STARTING"
	case StateRunning:
		return "RUNNING"
	case StateStopping:
		return "STOPPING"
	case StatePortCollision:
		return "PORT_COLLISION"
	case StateError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ButtonsEnabled returns whether Start/Stop/Restart actions are permitted.
func (s State) ButtonsEnabled() bool {
	return s != StateStarting && s != StateStopping
}
