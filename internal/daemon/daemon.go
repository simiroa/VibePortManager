//ff:what 헤드리스(GUI 없는) 데몬 모드: 배포/부팅 상주용 실행 경로
//ff:why PM2 대체 — Wails 창 없이 autostart 서버를 띄우고 crash 자동재시작까지 유지
package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/user/vpm/internal/config"
	"github.com/user/vpm/internal/logbuf"
	"github.com/user/vpm/internal/server"
	"github.com/user/vpm/internal/wslwrap"
	vmsys "github.com/user/vpm/pkg/syscall"
)

// Run starts VPM headless: it loads the saved config, launches every server
// marked autostart, keeps them alive (crash auto-restart is handled by the
// Manager), and blocks until SIGINT/SIGTERM — then stops all servers and exits.
// No Wails window, no tray. Logs go to stdout (server output still goes to the
// usual per-server log files).
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	mgr := server.NewManager()

	if b, err := vmsys.NewBackend("windows-native", ""); err == nil {
		mgr.RegisterBackend(b)
	}
	if distros, err := wslwrap.List(); err == nil {
		for _, d := range distros {
			if b, err := vmsys.NewBackend("wsl", d.Name); err == nil {
				mgr.RegisterBackend(b)
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onState := func(ev server.StateEvent) {
		logf("server %s → %s%s", ev.ServerID, ev.State, paren(ev.Error))
	}
	onCollision := func(ev server.CollisionEvent) {
		logf("collision: server %s port blocked by PID %d %s", ev.ServerID, ev.BlockingPID, paren(ev.BlockingDescription))
	}
	onLog := func(string, []string) {} // server output is already persisted to log files

	started := 0
	for _, proj := range cfg.Projects {
		for _, srv := range proj.Servers {
			if !srv.Autostart {
				continue
			}
			if err := mgr.Start(ctx, proj, srv, onState, onCollision, onLog); err != nil {
				logf("start %q failed: %v", srv.Name, err)
				continue
			}
			started++
		}
	}
	logf("daemon up: %d server(s) started across %d project(s)", started, len(cfg.Projects))
	if started == 0 {
		logf("note: no servers have autostart enabled — nothing to run")
	}

	// Hourly log rotation, same as the GUI.
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				for _, p := range cfg.Projects {
					_ = logbuf.Rotate(p.ID)
				}
			}
		}
	}()

	// Block until an OS shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logf("shutdown signal received — stopping servers…")
	cancel()
	for _, proj := range cfg.Projects {
		for _, srv := range proj.Servers {
			if mgr.GetState(srv.ID) == server.StateRunning {
				_ = mgr.Stop(context.Background(), proj, srv, onState)
			}
		}
	}
	logf("daemon stopped.")
	return nil
}

func logf(format string, a ...interface{}) {
	fmt.Printf("%s [vpm-daemon] "+format+"\n",
		append([]interface{}{time.Now().Format("15:04:05")}, a...)...)
}

func paren(s string) string {
	if s == "" {
		return ""
	}
	return "(" + s + ")"
}
