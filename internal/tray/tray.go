//ff:what Windows 시스템 트레이 아이콘 + 창 숨김/복원
//ff:why X 버튼 클릭 시 종료가 아닌 트레이 최소화 (plan.md §2.5)
package tray

// Manager controls the tray icon and show/hide window behaviour.
// Actual systray integration is done via Wails runtime hooks in app.go.
// This package owns the first-time warning state and autostart toggle.

// CloseAction describes what to do when the window close button is pressed.
type CloseAction int

const (
	ActionHideToTray CloseAction = iota // minimise to tray; show toast on first occurrence
	ActionQuit                          // actually quit (e.g. from tray menu "Quit")
)

// DecideCloseAction returns the action for the window close event.
// closeWarningSeen tracks whether the tray-toast has been shown before.
func DecideCloseAction(closeWarningSeen bool) CloseAction {
	return ActionHideToTray // always hide; caller handles first-time toast
}
