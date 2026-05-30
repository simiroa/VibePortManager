# VPM Known Limitations & Gap Analysis

**Date:** 2026-05-30  
**Reference spec:** `plan.md` (cross-platform functional spec)  
**Actual scope:** Windows 10/11 amd64 MVP

---

## Summary

`plan.md` specifies a cross-platform tool (Windows + macOS + Linux). The current implementation
covers Windows-only. macOS/Linux shell wrappers, process signals, and build targets are not
implemented. Several UI features from the spec were also deferred or simplified.

---

## §1 — Platform Scope

| Spec | Actual |
|---|---|
| Windows 10/11 (64-bit) | ✅ Implemented |
| macOS (Intel/Apple Silicon) | ❌ Not implemented |
| Linux (Ubuntu/Debian-based) | ❌ Not implemented |

`specs/manifest.yaml` records `darwin/*` as excluded from MVP scope. Linux is listed as
`secondary` (future/best-effort). No CI runner for non-Windows platforms.

---

## §2.1 — Shell Wrappers

**Windows** ✅  
`internal/pwshwrap/` builds the PowerShell invocation:
```
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "cmd /c <cmd>"
```
`internal/wslwrap/` handles WSL projects (detected via `\\wsl$\` or `\\wsl.localhost\` path prefix).

**macOS/Linux** ❌  
`/bin/zsh -l -c "<cmd>"` (login shell) not implemented. No `internal/loginshell/` package.
If the app were built for macOS/Linux, NVM/FNM/ASDF PATH injection would not work.

---

## §2.2 — Triple-Pass Port Killer

**Windows** ✅  
- Phase 1: `taskkill /T /PID <pid>` (process group)
- Phase 2: 3× 500ms TCP probe via `net.Listen` bind-then-close
- Phase 3: resolve blocker PID → `taskkill /F /PID <pid>`

**macOS/Linux** ❌  
- Phase 1: `syscall.Setpgid` + `SIGTERM` to `-PGID` — not implemented
- Phase 3: `lsof -t -i :<port>` + `kill -9 <pid>` — not implemented

---

## §2.4 — System Tray

**Implemented** ✅  
- Real tray icon via `getlantern/systray`, started in an OS-locked goroutine from
  `app.go:startup` (`internal/tray/systray.go`, panic-recovered, embeds `icon.ico`).
- Tray context menu: **Show VPM** (restores + un-minimises the window) and **Quit**
  (stops all running servers, then `runtime.Quit`).
- `beforeClose`: if servers are running **and** the tray is live (`tray.Active()`),
  the window hides to tray (first time emits `tray.firstHide` toast). If the tray failed
  to initialise, the close handler stops servers and exits normally — no unrecoverable
  hidden-window trap.

**Not implemented** ❌  
- Dynamic per-server Start/Stop entries in the tray menu (menu is static: Show / Quit).
- Live status indicators in the tray menu.

---

## §2.5 — Log Export

**Implemented** ✅  
- Logs written to `%APPDATA%/vpm/logs/<project_id>/<timestamp>_<uuid>.log`
- 7-day rotation + 100 MB per-project cap
- `[Export Logs]` button in the server-list view
- Native folder picker: `BrowseDirectory()` → `runtime.OpenDirectoryDialog`, then
  `ExportLogs(projectID, destPath)` copies the project's log files into the chosen folder.

**Deviation** ⚠️ (minor)  
plan.md says "Save File Dialog" (single-file). VPM uses a **directory** picker because a project
can have multiple log files; they are copied into the selected folder rather than saved as one file.

---

## §2.6 — Port Collision Wizard

**Implemented** ✅  
- `internal/ports/suggest.go`: `SuggestFree(base int)` scans `base+1..base+50`, then random ephemeral.
- `collision.detected` Wails event carries `suggestedFreePort`.
- `components/alert-panel.js`: inline alert strip shows collision info **and** a one-click
  **"Use free port :NNNN"** button. It dispatches `vpm:apply-suggested-port`, handled by
  `views/server-list.js:applySuggestedPort` → `UpdateServer` (persists new port) → `StartServer`.
- Per-port force kill is available in the **System Ports** panel (see below).

**Not implemented** ❌  
- A dedicated full-screen wizard flow (resolution is inline in the alert strip instead).

---

## §3 — Project Register / Server List

> **Note:** The 3-tab UI (Tab 1 Workspace / Tab 2 Dashboard / Tab 3 Logs) was replaced with a
> single unified server-list view driven by View Filter. All CRUD and log viewing happen in one
> component. References to "Tab 1/2/3" below map to the unified view.

**Implemented** ✅  
- Add Project (path + display name via modal)
- **Port auto-detection**: `AnalyzeProject(path)` runs static analysis on `.env*`, vite/next/nuxt
  config, and `--port` flags. Falls back to Detection Mode (runtime port snapshot diff) if static
  analysis finds no port.
- **package.json scripts detection**: `internal/project/analyze.go` picks `dev` > `start` >
  `serve` > `preview` priority and builds the full run command (e.g. `npm run dev`).
- Auto-detection of execution target (WSL vs Windows-native via path prefix)
- Package manager detection (bun > pnpm > yarn > npm, via lockfile scan)
- Add Server (name, command, port, autostart) — pre-filled from static analysis
- Start / Stop / Restart per server
- Remove project / server with confirm dialog
- Log strip integrated in server-list view (collapsible, auto-scroll)
- Export logs via native folder picker (`BrowseDirectory`)
- **Folder picker** (⊞ button in Add Project modal): native `OpenDirectoryDialog`.
- **System Port Analyzer + Port Killer + Registration**: the **System Ports** sidebar entry opens a
  panel (`components/system-ports.js`) that lists every listening port across Windows + WSL backends
  (`ScanSystemPorts` → port / PID / process / backend), tags VPM-managed ports, and offers a two-step
  inline **Kill** (`KillByPort`) plus **Rescan**. Clicking a row opens an **inline detail** (the scan
  list stays open — single-modal router: list → detail → add/new) where an unmanaged port can be
  **registered to an existing project or as a new project**. The start command is **auto-detected**
  from the listening process (`GetProcessCommand` → `Backend.ResolveProcessCommand`); leaving it blank
  registers a **monitor-only** server (VPM tracks the port but doesn't own the process — Start/Restart
  hidden, Stop still kills by port).
- **Light/Dark theme** (beyond plan.md): `theme.js` + titlebar toggle. Defaults to OS preference,
  manual override persisted in `localStorage`; CSS via `html:not(.dark)` overrides in `main.css`.

**Not implemented** ❌  
- **Drag-and-drop directory area**: a native folder picker is wired, but HTML5 drag-and-drop
  of a folder onto the window is not.

---

**Not implemented** ❌ (deferred)
- **Auto-Scroll Lock Toggle**: always auto-scrolls. No pause toggle.
- **Copy Logs to Clipboard**: no `[Copy]` button in log strip.

---

## Performance SLO Deviation

| Metric | plan.md | Revised SLO | Actual |
|---|---|---|---|
| Exe size | ≤ 15 MB | ≤ 15 MB | ~10 MB ✅ |
| Idle RAM | ≤ 30 MB | ≤ 80 MB | ~66 MB ✅ |
| Listening ports | 0 | 0 | 0 ✅ |

**RAM SLO**: 30 MB is unreachable with Wails v2 + WebView2 (Chromium renderer). The Go runtime
alone is ~5 MB; WebView2 host process adds ~60 MB baseline shared with other WebView2 consumers
system-wide. SLO revised to 80 MB in `specs/manifest.yaml`. Documented in `docs/build-report.md`.

**Status-bar RAM metric**: `GetSystemStats` now reports the VPM host process's real working-set
size (`mem_windows.go`, psapi `GetProcessMemoryInfo`), not the old `runtime.MemStats.Sys` Go-heap
figure. It still excludes the separate WebView2 child processes, so the status-bar number reflects
the Go/Wails host process only.

---

## What Would Be Needed to Close the Gaps

| Gap | Status | Notes |
|---|---|---|
| macOS/Linux shell wrapper | ❌ open | New `internal/loginshell/` pkg; CI on macOS/Linux runner |
| macOS/Linux port killer | ❌ open | Setpgid at spawn time; lsof fallback |
| Per-server tray menu entries | ❌ open | Static Show/Quit menu shipped; per-server Start/Stop not yet |
| Native folder picker for export | ✅ done | `BrowseDirectory` → `OpenDirectoryDialog` |
| Collision one-click resolve | ✅ done | Inline "Use free port" in alert strip; persists + relaunches |
| System Port Analyzer + killer | ✅ done | `components/system-ports.js` + `ScanSystemPorts`/`KillByPort` |
| Port backtrack → register | ✅ done | System Ports detail → add to project / new project (inline router, scan stays open) |
| Command auto-detect from port | ✅ done | `GetProcessCommand` → `Backend.ResolveProcessCommand` (win: CIM, wsl: /proc) |
| Monitor-only tracking (blank cmd) | ✅ done | empty command = track port only; `start.go` guard, UI hides Start/Restart |
| Light/Dark theme | ✅ done (beyond plan.md) | `theme.js` + titlebar toggle, OS-preference default, localStorage persist |
| System tray (restore/quit) | ✅ done | `getlantern/systray`, `internal/tray/systray.go` |
| Multi-server detect (PM2 import) | ✅ done | `internal/project/ecosystem.go` → `ProjectAnalysis.Servers` + multi-select import |
| Multi-server detect (general) | ✅ done | Detector framework (`detect.go`): workspaces(monorepo) + docker-compose + PM2 + single; per-type file |
| Monorepo workspaces | ✅ done | `detect_workspaces.go`: npm/yarn/pnpm/bun `workspaces` globs → per-app dev server, editable ports |
| Crash auto-restart | ✅ done | `Server.autorestart`; `watchExit` relaunch, 5×/60s crash-loop cap |
| Headless daemon (deploy) | ✅ done | `vpm --daemon` (`internal/daemon`); `--install-daemon` boot persistence |
| Port-drift re-detect | ✅ done (VPM-spawned only) | `ResyncServerPort` + `Backend.ResolveTreePort` (process tree ∩ netstat) |
| Logs of externally-started servers | ❌ not possible | OS can't attach to a foreign process's stdout — restart via VPM to capture |
| Drag-and-drop folder | ❌ open | Native picker wired; HTML5 folder drop not |
| Auto-scroll lock | ❌ open (trivial) | Toggle flag + conditional `scrollTop` |
| Copy to clipboard | ❌ open (trivial) | `navigator.clipboard.writeText()` |

### PM2 replacement stance (deploy)

VPM **does not depend on PM2 at runtime** — PM2 is AGPL-3.0, which would force VPM's own
source under AGPL on distribution / network use. Instead VPM **reimplements** the process
management it needs (start/stop/restart, logs, crash-restart, autostart, headless daemon)
and only **reads** `ecosystem.config.*` as a one-way import source. A config-file *format*
is data, not PM2 code, so parsing it carries no license obligation. Bundling/linking/shipping
the `pm2` package would; shelling out to a user-installed pm2 is the AGPL gray zone we avoid.
