# VPM Project State

## What This Is

Vibe Port Manager — Windows-only Wails v2 desktop app. Manages dev server ports/processes.
Go backend + Vanilla JS + Tailwind frontend. Status: **Windows MVP complete** (build/test/SSOT green).

> History of how each feature landed is in git. This file is the **current-state** reference:
> architecture, a feature index for navigation, design rules, and gates.

## Architecture

```
pkg/syscall/      ← Backend interface + per-target impls + MockExecutor
                    backend.go(interface: Spawn/SignalTree/KillPID/ResolvePortOwner/
                    ScanListenPorts/ResolveTreePort/ResolveProcessCommand),
                    win_backend.go(real, netstat/tasklist/taskkill/Toolhelp/CIM),
                    wsl_backend.go, linux_backend.go, *_stub.go(build-tag), mock.go
internal/
  portkiller/     ← Triple-Pass Port Killer (Graceful→Poll→Force)
  logbuf/         ← 100ms sliding log buffer + rotation (7-day / 100MB cap)
  pwshwrap/       ← PowerShell spawn spec builder
  wslwrap/        ← WSL distro list + bash spawn spec builder
  server/         ← Manager (state registry) + start/stop/restart/validate
                    start.go: monitor-only guard (empty command), crash auto-restart, watchExit
                    stop.go: reclaimPort (resolve→tree-kill→recheck loop, max 6 passes)
  ports/          ← ScanAll across backends + SuggestFree
  config/         ← JSON load/save (%APPDATA%/vpm/config.json), Project/Server structs
  project/        ← Add/remove + Detector framework (one type per file — see Detection below)
  daemon/         ← headless mode (vpm --daemon): GUI-less autostart + crash-restart
  tray/           ← System tray (getlantern/systray) + autostart registry (GUI + daemon)
app.go            ← Wails IPC surface (all methods real). mem_windows.go/mem_other.go: RAM metric
main.go           ← Wails bootstrap, Frameless window, beforeClose tray logic
frontend/src/
  main.js         ← shell layout + Wails event wiring + stats poller (5s) + boot (single view)
  state.js        ← UI state: view filter + selectedServerID + log line cache
  store.js        ← project/server state + lookups (findProjectByPort/findProjectByServer)
  wails.js        ← IPC + window-control + event helpers
  modal.js        ← Modal overlay (formContent + loading spinner). NO native alert/confirm
  theme.js        ← light/dark theme: OS-preference detect + toggle + localStorage persist
  components/
    titlebar.js   ← frameless dark titlebar: drag region + mini/min/max/close + theme toggle
    mini-bar.js   ← mini mode: active-port chips, 480×44 always-on-top bar
    sidebar.js    ← nav (All | projects | System Ports) + Add Project detection/import flow
    alert-panel.js← collision alert strip + one-click "use free port"
    server-card.js← PURE render of server card + compact list item (incl. monitor-only mode)
    log-strip.js  ← bottom log strip render + append/scroll helpers
    system-ports.js← System Ports inline router: list→detail→add/new (see Feature Index)
  views/
    server-list.js← unified filterable view: cards, CRUD modals, ⋮ card menu, log strip
specs/            ← SSOT: manifest.yaml, ipc.yaml, config.schema.json, state machines
```

## Feature Index (where to look)

| Feature | Entry point(s) |
|---|---|
| Start/Stop/Restart a server | `internal/server/{start,stop,restart}.go`; UI: `server-card.js` buttons → `server-list.js` handleAction |
| Server ⋮ menu (Autostart/Auto-restart toggle, Re-detect, Edit, Remove) | `server-list.js: _openCardMenu` (menu lives in `document.body`; click handler dispatches to `handleAction`) |
| Add Project + auto-detection | `sidebar.js` (flow) → `app.go: AnalyzeProject` → `internal/project` Detector framework |
| Multi-server import (monorepo/compose/PM2) | `sidebar.js: runMultiServerImport` ← `ProjectAnalysis.Servers` |
| Add/Edit Server modal (live port chips, Detect) | `server-list.js: addServerFlow/editServerFlow` |
| Port collision resolve | `alert-panel.js` "Use free port" → `server-list.js: applySuggestedPort` (`UpdateServer`+`StartServer`) |
| System Ports: scan / kill | `system-ports.js` (list view) → `ScanSystemPorts` / `KillByPort` |
| System Ports: backtrack → register to project / new project | `system-ports.js` detail→add/new (inline router; scan stays open) |
| Command auto-detect from a port | `system-ports.js: wireCmdField` → `GetProcessCommand` IPC → `Backend.ResolveProcessCommand` (win: CIM, wsl: /proc/<pid>/cmdline) |
| Monitor-only server (blank command) | register with empty command; `start.go` guards spawn; `server-card.js` hides Start/Restart, shows "external" badge |
| Port-drift re-detect (VPM-spawned only) | ⋮ "Re-detect port" → `ResyncServerPort` → `Backend.ResolveTreePort` (process tree ∩ netstat) |
| Crash auto-restart | `Server.autorestart`; `start.go: watchExit`+`registerRestart` (5×/60s crash-loop cap) |
| Headless daemon / boot persistence | `vpm --daemon` (`internal/daemon`); `--install-daemon` (Run-key) |
| Mini mode | `mini-bar.js` + `wails.js` window helpers |
| Light/Dark theme | `theme.js` + `titlebar.js` toggle; CSS via `html:not(.dark)` overrides in `main.css` |
| Log view / export | `log-strip.js` + `server-list.js`; `GetRecentLogs`/`ExportLogs` (native `BrowseDirectory`) |

## Project Detection (Detector framework)

`internal/project/detect.go`: `Detector` interface + two-stage `DetectAll` (primary → fallback when
primary finds nothing) with port/name dedupe. **One type per file** — add a detector + register in
`detectors()`. `analyze.go` aggregates single-app analysis + `DetectAll`. Ports extracted by
`detect_port.go`.

- **primary**: workspace (monorepo `workspaces`/pnpm-workspace globs), compose (docker-compose host
  ports, yaml.v3), pm2 (`ecosystem.config.*`), single (root app), python (Django/FastAPI/Flask), go
  (`go run`), rust (cargo), dotnet, ruby (Rails/Rack), php (Laravel/Symfony), java (Spring Boot),
  elixir (Phoenix), deno.
- **fallback** (only if primary empty): procfile (web/worker), task (Make/just/Taskfile dev/run/serve).
- PM detection (`detect_pm.go`): bun > pnpm > yarn > npm via lockfile (incl. `bun.lock`).

## Data Storage

User data in **`%APPDATA%\vpm\config.json`** (JSON, atomic write); logs in
`%APPDATA%\vpm\logs\<project_id>\`. Per-user APPDATA (not next to the exe) because Program Files is
read-only / must survive updates.

## Key Design Decisions

| Decision | Rule |
|---|---|
| No `alert()`/`confirm()` | Always use `Modal.*` from `modal.js` |
| OS calls via interface | `pkg/syscall.Backend` / `Executor` only — no direct `os/exec` in `internal/` |
| One file, one concept | filefunc rules in `internal/`; `app.go`/`main.go`/`frontend/` exempt |
| Button styles | Use `.btn-*` classes (`btn-primary/green/red/yellow/gray/ghost/disabled`) — never ad-hoc button styling |
| Port Killer phases | Graceful(SIGTERM) → Poll(3×500ms) → Force(resolve+kill or report) |
| Stop port reclaim | `reclaimPort`: resolve LISTEN owner → tree-kill (`taskkill /T /F`) → recheck, loop until free (≤6) |
| Cross-target block | Windows PID blocks WSL port → diagnostic only, no auto-kill |
| Log rotation | On startup + every 1h via goroutine; 7-day retention + 100MB cap |
| Monitor-only server | Empty command = track port only. `start.go` refuses to spawn; autostart skips it; UI hides Start/Restart |
| Command auto-detect | Best-effort fill from listening process; unknown → leave blank (monitor-only). Inline field validation, not modal errors |
| PM2 relation | **No runtime dependency** (PM2 = AGPL-3.0 → distribution taint). `ecosystem.config.*` parsed as import data only; process mgmt reimplemented |
| Crash restart | autorestart=true servers only; 60s window, 5-restart cap → ERROR (crash loop) |
| Daemon mode | `vpm --daemon` headless; boot residency = `--install-daemon` (separate Run-key value) |
| Logs of external servers | Not possible — OS can't attach to a foreign process's stdout. Restart via VPM to capture |
| Theme | Class-based (`darkMode: 'class'`); OS preference default, manual override persisted in localStorage |

## SLOs (from specs/manifest.yaml)

- exe ≤ 15MB (actual ~10 MB ✅)
- idle RAM ≤ 80MB (revised from 30 — Wails+WebView2 baseline ~65 MB irreducible; actual ~66 MB ✅)
- listening ports = 0 ✅

## Gates

```powershell
go build ./... ; go vet ./...
go test ./internal/... -coverprofile=coverage.out -timeout=30s   # portkiller/logbuf/pwshwrap ≥90%
go run ./scripts/validate-specs.go                                # SSOT cross-layer check
wails build -platform windows/amd64                               # (kill running exe first if locked)
```

## Known Gaps (deferred — Windows MVP scope)

Full analysis in `docs/known-limitations.md`. plan.md is cross-platform; this build is Windows-only.

- **macOS/Linux**: build targets, login shell, Setpgid/lsof port killer — not implemented.
- **Tray**: static Show/Quit only (no per-server entries).
- **Add Project**: native picker only (no HTML5 folder drag-drop); detected best script only (no full scripts dropdown).
- **Logs**: always auto-scrolls (no lock toggle); no copy-to-clipboard.

## Docs

- `docs/known-limitations.md` — gap analysis vs plan.md (kept current).
- `docs/build-report.md` — build metrics + RAM SLO rationale.
- `docs/adr/0001-unified-filterable-server-list.md` — why one view replaced the 3-tab UI.
- `docs/delegations/` — original agent task-split planning artifacts (historical; not maintained).
