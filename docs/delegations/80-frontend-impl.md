# Delegation: Frontend Implementation

> **Status: IMPLEMENTED (2026-05-28 session 4)**  
> This doc reflects the actual implementation. See `50-frontend.md` for architecture summary.

## Actual File Structure

```
frontend/src/
├── main.js              ← Shell HTML + event wiring + single-view boot (no tab router)
├── state.js             ← View Filter + selectedServerID + log cache (no activeView)
├── store.js             ← Project/server registry (re-exported via state.js)
├── wails.js             ← Thin IPC wrapper (all App.* methods)
├── modal.js             ← showModal(opts), showError, showConfirm
├── main.css             ← Tailwind + .btn-* / .badge-* custom classes
├── views/
│   └── server-list.js   ← Unified filterable view
└── components/
    ├── sidebar.js        ← All | project list | + Add Project
    └── alert-panel.js   ← Collision alert strip
```

## Wails IPC Methods (all implemented in app.go + wails.js)

```js
// Projects
ListProjects()
AddProject(path, name, targetOverride)
RemoveProject(id)

// Servers
AddServer(projectID, srv)         // srv = {id:'', name, command, port, autostart}
UpdateServer(projectID, srv)
RemoveServer(projectID, serverID)
StartServer(serverID)
StopServer(serverID)
RestartServer(serverID)

// Ports
ScanSystemPorts()                 // → PortEntry[]
KillByPort(port)
SuggestFreePort(startFrom)        // → int

// Logs
GetRecentLogs(serverID, maxLines) // → string[]
ExportLogs(projectID, destPath)

// WSL
ListWSLDistros()                  // → string[]

// Stats
GetSystemStats()
// → { ramMB: float, runningCount: int }

// Detection (NEW — for Add Project flow)
AnalyzeProject(path)
// → { port: int|null, command: string, scriptName: string, packageMgr: string }
GetListeningPorts()
// → int[]

// Settings
SetAutostart(enable)
```

## State API (`state.js`)

```js
// Re-exports from store.js:
getProjects(), setProjects(), getServerState(), applyStateChange(), findProjectByServer()

// UI state:
getSelectedServerID()           → string | null
setSelectedServer(id)           → emits vpm:server-selected

getProjectFilter()              → string | null  (projectID or null=All)
setProjectFilter(projectID)     → emits vpm:filter-changed

// Log cache (max 2000 lines per server):
getLogLines(serverID)           → string[]
appendLogLines(serverID, lines) → void
clearLogLines(serverID)         → void
```

**Note:** `getActiveView()` / `setActiveView()` were removed — no tab routing in this design.

## Event Bus

| Event | Emitter | Subscribers |
|---|---|---|
| `vpm:projects-updated` | `store.setProjects()` | sidebar, server-list |
| `vpm:state-changed` | `store.applyStateChange()` | sidebar, server-list |
| `vpm:server-selected` | `state.setSelectedServer()` | server-list (log strip) |
| `vpm:filter-changed` | `state.setProjectFilter()` | sidebar, server-list |
| `vpm:log-batch` | main.js (from Wails `server.log.batch`) | server-list (log strip) |
| `vpm:collision` | main.js (from Wails `collision.detected`) | alert-panel |

## CSS Class Conventions (defined in main.css)

| Class | Usage |
|---|---|
| `.btn` | Base button |
| `.btn-primary` | Primary action (indigo) |
| `.btn-ghost` | Cancel / secondary |
| `.btn-green` | Start |
| `.btn-red` | Stop / remove |
| `.btn-gray` | Neutral action |
| `.btn-disabled` | Disabled state |
| `.btn-xs` | Extra-small button (server card actions) |

## Constraints

- **No `alert()` / `confirm()`** — use `modal.js` (`showModal`, `showError`, `showConfirm`)
- **No framework** — vanilla JS ES2020 modules
- **No fetch/XHR** — all data via Wails IPC or events
- **No listening ports** — Wails IPC is OS-native pipe
