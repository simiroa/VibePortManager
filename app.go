package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/user/vpm/internal/config"
	"github.com/user/vpm/internal/logbuf"
	"github.com/user/vpm/internal/ports"
	"github.com/user/vpm/internal/project"
	"github.com/user/vpm/internal/server"
	"github.com/user/vpm/internal/tray"
	"github.com/user/vpm/internal/wslwrap"
	vmsys "github.com/user/vpm/pkg/syscall"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/google/uuid"
)

// App is the Wails binding struct. All exported methods become JS-callable.
// filefunc exception: multiple methods per file — Wails binding pattern.
type App struct {
	ctx     context.Context
	cfg     *config.Config
	mgr     *server.Manager
}

// NewApp creates the App singleton for Wails.
func NewApp() *App {
	return &App{mgr: server.NewManager()}
}

// startup is called by Wails after the window is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := config.Load()
	if err != nil {
		runtime.LogError(ctx, "config load: "+err.Error())
		cfg, _ = config.Load() // retry; use default
	}
	a.cfg = cfg

	// Register backends.
	winB, err := vmsys.NewBackend("windows-native", "")
	if err == nil {
		a.mgr.RegisterBackend(winB)
	}
	distros, _ := wslwrap.List()
	for _, d := range distros {
		wslB, err := vmsys.NewBackend("wsl", d.Name)
		if err == nil {
			a.mgr.RegisterBackend(wslB)
		}
	}

	// Rotate old logs once on startup, then every hour.
	go func() {
		for _, proj := range a.cfg.Projects {
			_ = logbuf.Rotate(proj.ID)
		}
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			for _, proj := range a.cfg.Projects {
				_ = logbuf.Rotate(proj.ID)
			}
		}
	}()

	// Autostart servers (skip monitor-only servers — they have no start command).
	for _, proj := range a.cfg.Projects {
		for _, srv := range proj.Servers {
			if srv.Autostart && strings.TrimSpace(srv.Command) != "" {
				a.startServer(proj, srv)
			}
		}
	}

	// Probe all registered servers: if a port is already occupied and the server
	// isn't already RUNNING/STARTING (e.g. from a previous VPM session that kept
	// processes alive after close), mark it RUNNING so the frontend shows the
	// correct state on boot. Autostart servers that hit PORT_COLLISION above are
	// also caught here.
	for _, proj := range a.cfg.Projects {
		for _, srv := range proj.Servers {
			cur := a.mgr.GetState(srv.ID)
			if cur == server.StateRunning || cur == server.StateStarting {
				continue // already handled
			}
			b := a.mgr.Backend(server.EffectiveTarget(proj))
			if b == nil {
				continue
			}
			if occupied, _ := server.IsPortOccupied(b, srv.Port); occupied {
				a.mgr.SyncState(srv.ID, server.StateRunning)
			}
		}
	}

	// System tray: provides the only way to restore a hidden window or quit
	// cleanly. Runs in its own OS-locked goroutine (see internal/tray).
	go tray.Run(tray.Handlers{
		OnShow: func() {
			runtime.WindowUnminimise(a.ctx)
			runtime.WindowShow(a.ctx)
		},
		OnQuit: func() {
			a.stopAllServers()
			runtime.Quit(a.ctx)
		},
	})
}

func (a *App) shutdown(_ context.Context) { tray.Stop() }

func (a *App) beforeClose(_ context.Context) bool {
	if !a.hasRunningServers() {
		return false // nothing running — close normally
	}

	// Servers are running. Hiding to tray is only safe if the tray icon is
	// live; otherwise a hidden window with no tray icon would be unrecoverable.
	if !tray.Active() {
		a.stopAllServers()
		return false // allow close — no trap
	}

	if !a.cfg.Settings.CloseWarningSeen {
		a.cfg.Settings.CloseWarningSeen = true
		config.Save(a.cfg)
		runtime.EventsEmit(a.ctx, "tray.firstHide")
	}
	runtime.WindowHide(a.ctx)
	return true // prevent close; reachable again via tray "Show VPM"
}

// hasRunningServers reports whether any registered server is RUNNING.
func (a *App) hasRunningServers() bool {
	for _, proj := range a.cfg.Projects {
		for _, srv := range proj.Servers {
			if a.mgr.GetState(srv.ID) == server.StateRunning {
				return true
			}
		}
	}
	return false
}

// stopAllServers stops every running server (best-effort) before the app exits.
func (a *App) stopAllServers() {
	for _, proj := range a.cfg.Projects {
		for _, srv := range proj.Servers {
			if a.mgr.GetState(srv.ID) == server.StateRunning {
				_ = a.mgr.Stop(a.ctx, proj, srv, a.emitState)
			}
		}
	}
}

// --- IPC methods (ipc.yaml 1:1) ---

// ListProjects returns all registered projects.
func (a *App) ListProjects() []config.Project {
	return a.cfg.Projects
}

// AddProject registers a new project from a path.
func (a *App) AddProject(path, displayName string, executionTargetOverride *string) (*config.Project, error) {
	id := uuid.NewString()
	result, err := project.Add(a.cfg, id, displayName, path)
	if err != nil {
		return nil, err
	}
	if executionTargetOverride != nil {
		for i, p := range a.cfg.Projects {
			if p.ID == result.Project.ID {
				a.cfg.Projects[i].ExecutionTarget = *executionTargetOverride
				result.Project.ExecutionTarget = *executionTargetOverride
				break
			}
		}
		config.Save(a.cfg)
	}
	return &result.Project, nil
}

// RemoveProject removes a project and stops its servers.
func (a *App) RemoveProject(id string) error {
	for _, proj := range a.cfg.Projects {
		if proj.ID == id {
			for _, srv := range proj.Servers {
				_ = a.mgr.Stop(a.ctx, proj, srv, a.emitState)
			}
			break
		}
	}
	return project.Remove(a.cfg, id)
}

// ListWSLDistros returns names of installed WSL distributions.
func (a *App) ListWSLDistros() []string {
	distros, _ := wslwrap.List()
	names := make([]string, len(distros))
	for i, d := range distros {
		names[i] = d.Name
	}
	return names
}

// AddServer adds a new server definition to a project.
func (a *App) AddServer(projectID string, srv config.Server) (*config.Server, error) {
	srv.ID = uuid.NewString()
	for i, p := range a.cfg.Projects {
		if p.ID == projectID {
			if err := server.ValidatePortUnique(p, "", srv.Port); err != nil {
				return nil, err
			}
			a.cfg.Projects[i].Servers = append(a.cfg.Projects[i].Servers, srv)
			return &srv, config.Save(a.cfg)
		}
	}
	return nil, fmt.Errorf("project not found: %s", projectID)
}

// UpdateServer replaces a server definition (port change requires stop first).
func (a *App) UpdateServer(projectID string, srv config.Server) error {
	for i, p := range a.cfg.Projects {
		if p.ID != projectID {
			continue
		}
		if err := server.ValidatePortUnique(p, srv.ID, srv.Port); err != nil {
			return err
		}
		for j, s := range p.Servers {
			if s.ID == srv.ID {
				a.cfg.Projects[i].Servers[j] = srv
				return config.Save(a.cfg)
			}
		}
		return fmt.Errorf("server not found: %s", srv.ID)
	}
	return fmt.Errorf("project not found: %s", projectID)
}

// RemoveServer stops and removes a server.
func (a *App) RemoveServer(projectID, serverID string) error {
	for i, p := range a.cfg.Projects {
		if p.ID != projectID {
			continue
		}
		for _, s := range p.Servers {
			if s.ID == serverID {
				_ = a.mgr.Stop(a.ctx, p, s, a.emitState)
				break
			}
		}
		servers := make([]config.Server, 0, len(p.Servers))
		for _, s := range p.Servers {
			if s.ID != serverID {
				servers = append(servers, s)
			}
		}
		a.cfg.Projects[i].Servers = servers
		return config.Save(a.cfg)
	}
	return fmt.Errorf("project not found: %s", projectID)
}

// StartServer starts the server process.
func (a *App) StartServer(serverID string) error {
	proj, srv, err := a.findServer(serverID)
	if err != nil {
		return err
	}
	return a.startServer(proj, srv)
}

func (a *App) startServer(proj config.Project, srv config.Server) error {
	return a.mgr.Start(a.ctx, proj, srv, a.emitState, a.emitCollision, a.emitLog)
}

// StopServer stops the server via Triple-Pass Port Killer.
func (a *App) StopServer(serverID string) error {
	proj, srv, err := a.findServer(serverID)
	if err != nil {
		return err
	}
	return a.mgr.Stop(a.ctx, proj, srv, a.emitState)
}

// RestartServer stops then starts the server.
func (a *App) RestartServer(serverID string) error {
	proj, srv, err := a.findServer(serverID)
	if err != nil {
		return err
	}
	return a.mgr.Restart(a.ctx, proj, srv, a.emitState, a.emitCollision, a.emitLog)
}

// ScanSystemPorts returns all listening ports across all backends.
func (a *App) ScanSystemPorts() ([]ports.PortEntry, error) {
	return ports.ScanAll(a.mgr.AllBackends())
}

// KillByPort kills whoever is listening on port.
func (a *App) KillByPort(port int) error {
	for _, b := range a.mgr.AllBackends() {
		owner, err := b.ResolvePortOwner(port)
		if err != nil || owner.PID == 0 {
			continue
		}
		return b.KillPID(owner.PID)
	}
	return fmt.Errorf("no process found on port %d", port)
}

// GetProcessCommand returns the full command line of the process listening on a
// scanned port, used to auto-fill the start command when registering it. The
// backendID selects the right namespace (Windows vs WSL PIDs differ); an empty
// backendID falls back to the windows-native backend. Returns "" when unknown.
func (a *App) GetProcessCommand(pid int, backendID string) (string, error) {
	if pid <= 0 {
		return "", nil
	}
	for _, b := range a.mgr.AllBackends() {
		if backendID == "" || b.ID() == backendID {
			cmd, err := b.ResolveProcessCommand(pid)
			if err != nil {
				return "", nil // unknown — caller falls back to monitor-only
			}
			return cmd, nil
		}
	}
	return "", nil
}

// SuggestFreePort returns a free port near startFrom.
func (a *App) SuggestFreePort(startFrom int) int {
	if len(a.mgr.AllBackends()) == 0 {
		return startFrom
	}
	p, err := ports.SuggestFree(a.mgr.AllBackends()[0], startFrom)
	if err != nil {
		return startFrom
	}
	return p
}

// GetRecentLogs returns up to maxLines recent log lines from the server's current log file.
// Lines are pushed live via "server.log.batch" events; this is for initial hydration on tab open.
func (a *App) GetRecentLogs(serverID string, maxLines int) []string {
	path := a.mgr.LogPath(serverID)
	if path == "" {
		return []string{}
	}
	lines, err := tailFile(path, maxLines)
	if err != nil {
		runtime.LogError(a.ctx, "GetRecentLogs: "+err.Error())
		return []string{}
	}
	return lines
}

// ExportLogs copies all log files for a project into destPath (creates dir if needed).
func (a *App) ExportLogs(projectID, destPath string) error {
	srcDir, err := logbuf.LogDir(projectID)
	if err != nil {
		return fmt.Errorf("log dir: %w", err)
	}
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read log dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(destPath, e.Name())
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", e.Name(), err)
		}
	}
	return nil
}

// GetSystemStats returns VPM RAM usage and running server count for the status bar.
// Also re-probes STOPPED/ERROR servers every call so externally-started processes
// are reflected within the next 5 s poll cycle without requiring a VPM restart.
func (a *App) GetSystemStats() map[string]interface{} {
	running := 0
	for _, p := range a.cfg.Projects {
		for _, s := range p.Servers {
			st := a.mgr.GetState(s.ID)
			if st == server.StateStopped || st == server.StateError {
				b := a.mgr.Backend(server.EffectiveTarget(p))
				if b != nil {
					if occupied, _ := server.IsPortOccupied(b, s.Port); occupied {
						a.mgr.SyncState(s.ID, server.StateRunning)
						a.emitState(server.StateEvent{ServerID: s.ID, State: server.StateRunning})
						st = server.StateRunning
					}
				}
			}
			if st == server.StateRunning {
				running++
			}
		}
	}
	// Prefer the OS-reported working set (real physical RAM); fall back to the
	// Go heap estimate only if the platform call is unavailable.
	ramMB := procWorkingSetMB()
	if ramMB == 0 {
		var m goruntime.MemStats
		goruntime.ReadMemStats(&m)
		ramMB = float64(m.Sys) / 1024 / 1024
	}
	return map[string]interface{}{
		"ramMB":           ramMB,
		"runningCount":    running,
		"activePortCount": running,
	}
}

// AnalyzeProject performs static analysis on a project directory.
// Returns detected port, suggested command, and package manager.
// If Port is nil, the frontend should enter Detection Mode.
func (a *App) AnalyzeProject(path string) project.ProjectAnalysis {
	return project.Analyze(path)
}

// GetListeningPorts returns all distinct port numbers currently listening on the system.
// Used by the frontend for Detection Mode: snapshot before → scan after → diff = candidates.
func (a *App) GetListeningPorts() ([]int, error) {
	entries, err := ports.ScanAll(a.mgr.AllBackends())
	if err != nil {
		return nil, err
	}
	seen := make(map[int]bool, len(entries))
	result := make([]int, 0, len(entries))
	for _, e := range entries {
		if !seen[e.Port] {
			seen[e.Port] = true
			result = append(result, e.Port)
		}
	}
	return result, nil
}

// GetAllServerStates returns a serverID → state-string map for every registered server.
// Called by the frontend on boot to sync initial state without waiting for events.
func (a *App) GetAllServerStates() map[string]string {
	result := make(map[string]string)
	for _, p := range a.cfg.Projects {
		for _, s := range p.Servers {
			result[s.ID] = a.mgr.GetState(s.ID).String()
		}
	}
	return result
}

// BrowseDirectory opens a native folder-picker dialog and returns the selected path.
// Returns an empty string if the user cancels.
func (a *App) BrowseDirectory() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Project Folder",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// ResyncServerPort re-detects the actual LISTEN port of a VPM-spawned server
// whose port drifted from the configured value (e.g. dev server fell back to a
// different port). Only works for servers VPM started — for externally-started
// servers VPM holds no handle, so restart it via VPM first. Returns the current
// port (unchanged if already correct) or an error.
func (a *App) ResyncServerPort(serverID string) (int, error) {
	proj, srv, err := a.findServer(serverID)
	if err != nil {
		return 0, err
	}
	pid, ok := a.mgr.HandlePID(serverID)
	if !ok || pid == 0 {
		return 0, fmt.Errorf("re-detect only works for servers VPM started — restart this server from VPM first")
	}
	b := a.mgr.Backend(server.EffectiveTarget(proj))
	if b == nil {
		return 0, fmt.Errorf("no backend for %s", proj.ExecutionTarget)
	}
	port, err := b.ResolveTreePort(pid)
	if err != nil {
		return 0, err
	}
	if port == 0 {
		return 0, fmt.Errorf("no listening port found for this server's process")
	}
	if port == srv.Port {
		return port, nil // already correct
	}
	for i, p := range a.cfg.Projects {
		if p.ID != proj.ID {
			continue
		}
		for j, s := range p.Servers {
			if s.ID == serverID {
				a.cfg.Projects[i].Servers[j].Port = port
				if err := config.Save(a.cfg); err != nil {
					return 0, err
				}
				a.mgr.SyncState(serverID, server.StateRunning)
				a.emitState(server.StateEvent{ServerID: serverID, State: server.StateRunning})
				return port, nil
			}
		}
	}
	return port, nil
}

// SetAutostart toggles VPM Windows autostart.
func (a *App) SetAutostart(enable bool) error {
	if err := tray.SetAutostart(enable); err != nil {
		return err
	}
	a.cfg.Settings.AutostartVPM = enable
	return config.Save(a.cfg)
}

// --- helpers ---

func (a *App) findServer(serverID string) (config.Project, config.Server, error) {
	for _, p := range a.cfg.Projects {
		for _, s := range p.Servers {
			if s.ID == serverID {
				return p, s, nil
			}
		}
	}
	return config.Project{}, config.Server{}, fmt.Errorf("server not found: %s", serverID)
}

func (a *App) emitState(ev server.StateEvent) {
	runtime.EventsEmit(a.ctx, "server.state.changed", map[string]interface{}{
		"serverID": ev.ServerID,
		"state":    ev.State.String(),
		"error":    ev.Error,
	})
}

func (a *App) emitCollision(ev server.CollisionEvent) {
	runtime.EventsEmit(a.ctx, "collision.detected", map[string]interface{}{
		"serverID":            ev.ServerID,
		"origin":              ev.Origin,
		"blockingPID":         ev.BlockingPID,
		"blockingDescription": ev.BlockingDescription,
		"suggestedFreePort":   ev.SuggestedFreePort,
	})
}

func (a *App) emitLog(serverID string, lines []string) {
	runtime.EventsEmit(a.ctx, "server.log.batch", map[string]interface{}{
		"serverID": serverID,
		"lines":    lines,
	})
}

// tailFile returns the last n lines of the file at path.
func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(lines) <= n {
		return lines, nil
	}
	return lines[len(lines)-n:], nil
}

// copyFile copies src to dst, creating dst if it does not exist.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
