// state.js — unified UI state: extends store.js with selection, filter, and log cache
// Components import from here, not directly from store.js.

export * from './store.js'

const emit = (type, detail) => window.dispatchEvent(new CustomEvent(type, { detail }))

// ── Server selection (for log strip) ─────────────────────────────────────────
let _selectedServerID = null

export function getSelectedServerID() { return _selectedServerID }

export function setSelectedServer(id) {
  _selectedServerID = id || null
  emit('vpm:server-selected', { serverID: _selectedServerID })
}

// ── View Filter (null = All, project_id = specific project) ──────────────────
let _activeProjectFilter = null

export function getProjectFilter() { return _activeProjectFilter }

export function setProjectFilter(projectID) {
  _activeProjectFilter = projectID || null
  emit('vpm:filter-changed', { projectID: _activeProjectFilter })
}

// ── Log line cache (max 2000 lines per server) ────────────────────────────────
const _logLines = new Map() // serverID → string[]
const MAX_LOG_LINES = 2000

export function getLogLines(serverID) {
  return _logLines.get(serverID) ?? []
}

export function appendLogLines(serverID, lines) {
  if (!_logLines.has(serverID)) _logLines.set(serverID, [])
  const arr = _logLines.get(serverID)
  arr.push(...lines)
  if (arr.length > MAX_LOG_LINES) arr.splice(0, arr.length - MAX_LOG_LINES)
}

export function clearLogLines(serverID) {
  _logLines.set(serverID, [])
}
