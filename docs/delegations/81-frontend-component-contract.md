# 81 — Frontend Component Contract

> **Status: IMPLEMENTED (2026-05-28 session 4)**  
> Reflects actual implementation. Old 3-tab contracts (port-dashboard, project-register, log-monitor, log-panel) are superseded.

---

## App Shell Layout (actual)

```html
<body>
  <div id="modal-overlay">           ← modal.js

  <div class="flex h-screen">
    <aside>                          ← sidebar.js
      <div id="sidebar-shell">       ← All + project list + Add Project
      <div id="status-bar">          ← stat-running, stat-ram
    </aside>

    <div class="flex-1 flex flex-col">
      <div id="alert-panel">         ← alert-panel.js (hidden by default)
      <main id="view-content">       ← server-list.js (always mounted, never swapped)
    </div>
  </div>
</body>
```

**No `#log-panel` in shell.** Log strip is the bottom section inside `server-list.js`.

---

## Sidebar (`components/sidebar.js`)

```
init(sidebarEl)   ← single element
```

Renders into `sidebarEl`:
1. "⚡ All Servers" → `setProjectFilter(null)`
2. Project list — each row: status dot + name + server count → `setProjectFilter(id)` (toggle)
3. "+ Add Project" → `runAddProjectFlow()` (3-step detection modal)

Re-renders on: `vpm:projects-updated`, `vpm:state-changed`, `vpm:filter-changed`.

**No nav items. No server chips. No app title. No settings link.**

---

## Server List View (`views/server-list.js`)

```
init(container)   ← <main id="view-content">
```

Single always-mounted view. Internal state: `_searchQuery`, `_logExpanded`, `_logPendingScroll`.

**Sections:**

1. **TopBar** — varies by View Filter:
   - `null`: search input (filters by project/server name or port)
   - `project_id`: project name + path + "Restart All" + "Remove Project" button

2. **Server cards** — filtered/searched, grouped by project in All view:
   - State badge + spinner for STARTING/STOPPING
   - Start / Stop / Restart buttons (state-aware, disabled when busy)
   - Edit + Remove buttons
   - Click card → `setSelectedServer(id)`, auto-open log strip, hydrate via `GetRecentLogs`

3. **Log strip** (bottom, collapsible):
   - Header always visible (click = toggle expand/collapse)
   - Collapsed: header only | Expanded: `h-44` scrollable log view
   - Shows `getLogLines(selectedServerID)` — appended on `vpm:log-batch`
   - Buttons: Clear, Export… (stop propagation)

Re-renders on: `vpm:filter-changed`, `vpm:projects-updated`, `vpm:state-changed`, `vpm:server-selected`, `vpm:log-batch`.

---

## Alert Panel (`components/alert-panel.js`)

```
init(el)   ← <div id="alert-panel">
```

- Hidden by default
- Listen `vpm:collision` → show alert strip with PID + description + origin
- Dismiss button → hide

---

## State API (`state.js`)

```js
getSelectedServerID() / setSelectedServer(id)    → emits vpm:server-selected
getProjectFilter()    / setProjectFilter(id)     → emits vpm:filter-changed
getLogLines(serverID) / appendLogLines / clearLogLines
// store.js re-exports: getProjects, setProjects, getServerState, applyStateChange
```

**Removed:** `getActiveView()`, `setActiveView()` — no tab routing.

---

## View Module Contract

```js
export async function init(container: HTMLElement): Promise<(() => void) | void>
```

- `container` = `<main id="view-content">`
- Only one view: `server-list.js`. Never swapped.

---

## Do NOT

- Add `alert()` / `confirm()` — use `modal.js`
- Import from deleted files: `views/port-dashboard.js`, `views/project-register.js`, `views/log-monitor.js`, `components/log-panel.js`
- Call `setActiveView()` — removed
- Add framework JS
- Use `tabs/` path prefix — was never created
