# 50. Frontend — Unified Server-List UX

> **Status: IMPLEMENTED (2026-05-28 session 4)**  
> This doc reflects the actual implemented design, not the original 3-tab plan.md spec.  
> Original plan.md §3 described 3 separate tabs — that approach was superseded.

## Architecture

Single filterable view. No tab router. View Filter (`null` = All, `project_id` = project detail) drives rendering inside one `server-list.js` component.

```
frontend/
├── src/
│   ├── main.js              # Shell layout + event wiring + single-view boot
│   ├── state.js             # View Filter + selectedServerID + log cache
│   ├── store.js             # Project/server state (keep for compat)
│   ├── wails.js             # IPC thin wrapper
│   ├── modal.js             # Modal overlay (showModal, showError, showConfirm)
│   ├── main.css             # @tailwind + .btn-* / .badge-* custom classes
│   ├── views/
│   │   └── server-list.js   # Unified filterable view (ALL + project detail + log strip)
│   └── components/
│       ├── sidebar.js       # All | project list | + Add Project (with detection flow)
│       └── alert-panel.js   # Collision alert strip
└── index.html
```

**Deleted (superseded):**
- `views/port-dashboard.js`, `views/project-register.js`, `views/log-monitor.js`
- `components/log-panel.js`

## Shell Layout

```
┌───────────────────────────────────────────────┐
│  sidebar (w-52)          │  view-content       │
│  ⚡ All Servers           │  [TopBar]           │
│  ● my-app        2       │  [Server cards]     │
│  ○ auth-svc      1       │  ...                │
│  + Add Project           │  [Log strip ▼]      │
│  ──────────────────────  │                     │
│  3 running / 66 MB RAM   │                     │
└───────────────────────────────────────────────┘
```

## View Filter Behaviour

| Filter | TopBar | Server list | Log strip |
|---|---|---|---|
| `null` (All) | search input | all projects, grouped | selected server's logs |
| `project_id` | project name + path + "Restart All" + "Remove" | that project's servers only | selected server's logs |

Sidebar click: `setProjectFilter(id)` — toggle (click again → back to All).

## Add Project Flow (Detection Mode)

3-step modal sequence in `sidebar.js`:

1. **Path + display name** form
2. **Static analysis** (`AnalyzeProject(path)`) → if port found, skip to step 3
   - If `port === null`: **Detection Mode** — snapshot ports → user starts server → re-scan → diff → pick candidate
3. **Server Proposal** — pre-filled name/command/port/autostart → `AddProject` + `AddServer`

Static analysis priority: `.env.local` → `.env` → vite/next/nuxt config → `--port` flag in command.
Package manager detection: bun > pnpm > yarn > npm (by lockfile).

## IPC Methods Used

All in `wails.js` thin wrapper:

```js
ListProjects, AddProject, RemoveProject
AddServer, UpdateServer, RemoveServer
StartServer, StopServer, RestartServer
GetRecentLogs, ExportLogs, GetSystemStats
AnalyzeProject    // NEW: static port/command analysis
GetListeningPorts // NEW: for Detection Mode snapshot/scan
SetAutostart
```

## Wails Events

| Wails event | Handler |
|---|---|
| `server.state.changed` | `store.applyStateChange()` → `vpm:state-changed` |
| `server.log.batch` | `state.appendLogLines()` → `vpm:log-batch` |
| `collision.detected` | `vpm:collision` → alert-panel |
| `tray.firstHide` | showModal "Minimised to Tray" |

## Native alert() — BANNED

Zero `window.alert` / `window.confirm` / `window.prompt`. Use `modal.js` only.
