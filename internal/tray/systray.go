//ff:what 시스템 트레이 아이콘 + Show/Quit 메뉴 (getlantern/systray)
//ff:why X 버튼으로 창을 숨겼을 때 복귀/종료 수단 제공 — 트레이 없으면 사용자가 갇힘 (plan.md §2.4)
package tray

import (
	_ "embed"
	"runtime"
	"sync/atomic"

	"github.com/getlantern/systray"
)

//go:embed icon.ico
var iconData []byte

// running is 1 while the tray icon is live. Read via Active().
var running int32

// Handlers wires tray menu clicks back to the app.
type Handlers struct {
	OnShow func() // user clicked "Show VPM"
	OnQuit func() // user clicked "Quit" (run before the process exits)
}

// Active reports whether the tray icon is currently live. The close handler
// uses this to decide whether hiding the window is safe (a hidden window with
// no tray icon would be unrecoverable).
func Active() bool { return atomic.LoadInt32(&running) == 1 }

// Run starts the tray event loop. It BLOCKS until Stop()/systray.Quit(), so
// callers must invoke it in a dedicated goroutine. The goroutine is locked to
// its OS thread because the Win32 message pump must run on the thread that
// created the window. A panic in tray init is recovered so it can never crash
// the host app — Active() simply stays false and the close handler falls back.
func Run(h Handlers) {
	defer func() {
		_ = recover()
		atomic.StoreInt32(&running, 0)
	}()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	onReady := func() {
		systray.SetIcon(iconData)
		systray.SetTitle("Vibe Port Manager")
		systray.SetTooltip("Vibe Port Manager")

		mShow := systray.AddMenuItem("Show VPM", "Restore the VPM window")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Stop all servers and exit")

		atomic.StoreInt32(&running, 1)

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					if h.OnShow != nil {
						h.OnShow()
					}
				case <-mQuit.ClickedCh:
					if h.OnQuit != nil {
						h.OnQuit()
					}
					systray.Quit()
					return
				}
			}
		}()
	}

	onExit := func() { atomic.StoreInt32(&running, 0) }

	systray.Run(onReady, onExit)
}

// Stop tears down the tray icon. Safe to call multiple times.
func Stop() { systray.Quit() }
