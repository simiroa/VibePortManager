// store.js — simple reactive store for project + server state
// No framework: uses custom event dispatch on state mutations.

let _projects = []
const _serverStates     = new Map() // serverID → State string
const _serverStartTimes = new Map() // serverID → Date.now() when RUNNING started
const _serverErrors     = new Map() // serverID → last error string

const emit = (type, detail) => window.dispatchEvent(new CustomEvent(type, { detail }))

export function getProjects()           { return _projects }
export function getServerState(id)      { return _serverStates.get(id) ?? 'STOPPED' }
export function getServerStartTime(id)  { return _serverStartTimes.get(id) ?? null }
export function getServerError(id)      { return _serverErrors.get(id) ?? null }

export function setProjects(projects) {
  _projects = projects
  emit('vpm:projects-updated', projects)
}

export function applyStateChange({ serverID, state, error }) {
  _serverStates.set(serverID, state)
  if (state === 'RUNNING') {
    _serverStartTimes.set(serverID, Date.now())
    _serverErrors.delete(serverID)
  } else if (state === 'ERROR' || state === 'PORT_COLLISION') {
    _serverStartTimes.delete(serverID)
    if (error) _serverErrors.set(serverID, error)
  } else if (state !== 'STARTING') {
    _serverStartTimes.delete(serverID)
    _serverErrors.delete(serverID)
  }
  emit('vpm:state-changed', { serverID, state })
}

export function findProjectByServer(serverID) {
  for (const p of _projects) {
    for (const s of p.servers ?? []) {
      if (s.id === serverID) return { project: p, server: s }
    }
  }
  return null
}
