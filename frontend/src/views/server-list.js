// views/server-list.js — unified filterable server list (orchestration layer)
// Renders project sections + wires all server actions.
// Card rendering → components/server-card.js
// Log strip      → components/log-strip.js

import { getProjects, getServerState, setProjects, findProjectByServer } from '../store.js'
import {
  getProjectFilter, setProjectFilter,
  getSelectedServerID, setSelectedServer,
  getLogLines, appendLogLines, clearLogLines,
} from '../state.js'
import {
  StartServer, StopServer, RestartServer, ResyncServerPort,
  AddServer, RemoveServer, RemoveProject,
  UpdateServer, SuggestFreePort, BrowserOpenURL,
  GetRecentLogs, ListProjects, ExportLogs, GetListeningPorts,
  BrowseDirectory,
} from '../wails.js'
import { showModal, showError, showConfirm } from '../modal.js'
import { renderServerCard, renderServerListItem, renderAddServerCard } from '../components/server-card.js'
import { renderLogStrip, appendLogLinesDOM, scrollLogToBottom } from '../components/log-strip.js'

// ── Module state ──────────────────────────────────────────────────────────────

let _el = null
let _searchQuery      = ''
let _logExpanded      = false
let _logPendingScroll = false
let _logNewCount      = 0   // lines received while strip is hidden/collapsed (item 5)
let _uptimeInterval   = null

// ── Entry point ───────────────────────────────────────────────────────────────

export async function init(container) {
  _el = container
  _el.className = 'flex flex-col h-full'

  // Delegated click handler — attached ONCE on the stable container.
  // (render() replaces innerHTML, so child listeners are re-wired per render,
  // but this container listener must not be re-added or it would stack.)
  _el.addEventListener('click', handleAction)

  const handlers = {
    filterChanged:   () => render(),
    projectsUpdated: () => render(),
    stateChanged:    () => render(),
    serverSelected:  () => render(),
    logBatch:        onLogBatch,
    applyPort:       e => applySuggestedPort(e.detail),
  }

  window.addEventListener('vpm:filter-changed',        handlers.filterChanged)
  window.addEventListener('vpm:projects-updated',      handlers.projectsUpdated)
  window.addEventListener('vpm:state-changed',         handlers.stateChanged)
  window.addEventListener('vpm:server-selected',       handlers.serverSelected)
  window.addEventListener('vpm:log-batch',             handlers.logBatch)
  window.addEventListener('vpm:apply-suggested-port',  handlers.applyPort)

  // Refresh uptime badges every 60 s (item 4)
  _uptimeInterval = setInterval(() => render(), 60000)

  render()

  return function cleanup() {
    window.removeEventListener('vpm:filter-changed',        handlers.filterChanged)
    window.removeEventListener('vpm:projects-updated',      handlers.projectsUpdated)
    window.removeEventListener('vpm:state-changed',         handlers.stateChanged)
    window.removeEventListener('vpm:server-selected',       handlers.serverSelected)
    window.removeEventListener('vpm:log-batch',             handlers.logBatch)
    window.removeEventListener('vpm:apply-suggested-port',  handlers.applyPort)
    if (_uptimeInterval) clearInterval(_uptimeInterval)
  }
}

// ── Render ────────────────────────────────────────────────────────────────────

function render() {
  if (!_el) return
  const filter = getProjectFilter()

  _el.innerHTML = `
    <div class="flex-1 overflow-y-auto min-h-0">
      ${renderTopBar(filter)}
      ${renderProjects(filter)}
    </div>
    ${renderLogStrip(_logExpanded, _logNewCount)}
  `
  wireEvents()
  if (_logPendingScroll) {
    scrollLogToBottom(_el)
    _logPendingScroll = false
  }
}

function renderTopBar(filter) {
  if (filter === null) {
    return `
      <div class="px-4 pt-4 pb-2">
        <input id="sl-search" type="text" value="${esc(_searchQuery)}"
          placeholder="Search servers…"
          class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm
                 text-gray-100 placeholder-gray-600 focus:outline-none focus:border-indigo-500">
      </div>
    `
  }
  const proj = getProjects().find(p => p.id === filter)
  if (!proj) return ''
  return `
    <div class="px-4 pt-4 pb-2 flex items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <h2 class="text-sm font-semibold text-gray-100 truncate">${esc(proj.name)}</h2>
        <button data-action="copy-path" data-path="${esc(proj.path ?? '')}"
          title="Click to copy path"
          class="mono text-xs text-gray-500 hover:text-indigo-400 truncate mt-0.5 cursor-pointer
                 max-w-full transition-colors hover:bg-gray-800/50 px-1.5 py-0.5 rounded">
          ${esc(proj.path ?? '')}
        </button>
        ${proj.packageManager && proj.packageManager !== 'none'
          ? `<span class="inline-block mt-1 text-xs text-indigo-400 bg-indigo-900/40 rounded px-1.5 py-0.5">${esc(proj.packageManager)}</span>`
          : ''}
      </div>
      <div class="flex gap-1.5 shrink-0 mt-0.5">
        <button data-action="restart-all" data-project-id="${esc(proj.id)}"
          class="text-xs px-2 py-1 rounded bg-gray-800 text-gray-300 hover:bg-gray-700 hover:text-white transition-colors">
          Restart All
        </button>
        <button data-action="remove-project" data-project-id="${esc(proj.id)}"
          class="text-xs px-2 py-1 rounded bg-gray-800 text-red-400 hover:bg-red-900/40 hover:text-red-300 transition-colors">
          Remove
        </button>
      </div>
    </div>
  `
}

function renderProjects(filter) {
  const all = getProjects()
  const projects = filter !== null
    ? all.filter(p => p.id === filter)
    : all.filter(p => {
        if (!_searchQuery) return true
        const q = _searchQuery.toLowerCase()
        if (p.name.toLowerCase().includes(q)) return true
        return (p.servers ?? []).some(s =>
          s.name.toLowerCase().includes(q) ||
          String(s.port).includes(q) ||
          (s.command ?? '').toLowerCase().includes(q)
        )
      })

  if (projects.length === 0) {
    return all.length === 0
      ? `<div class="flex flex-col items-center justify-center h-48 gap-3 text-center px-4">
          <p class="text-sm text-gray-500">No projects yet.</p>
          <p class="text-xs text-gray-600">Add your first project to start managing dev servers.</p>
          <button data-action="add-project"
            class="mt-1 px-4 py-2 rounded-lg bg-indigo-700 hover:bg-indigo-600 text-white text-sm font-medium transition-colors">
            + Add First Project
          </button>
        </div>`
      : `<p class="px-4 py-6 text-sm text-gray-600">No results.</p>`
  }

  return `<div class="px-4 pb-4 space-y-6">
    ${projects.map(p => renderProjectSection(p, filter)).join('')}
  </div>`
}

function renderProjectSection(proj, filter) {
  const servers = proj.servers ?? []

  const header = filter === null ? `
    <div class="flex items-center justify-between mb-2">
      <button data-action="goto-project" data-project-id="${esc(proj.id)}"
        class="text-xs font-semibold text-gray-400 hover:text-indigo-300 transition-colors truncate">
        ${esc(proj.name)}
      </button>
    </div>
  ` : ''

  const cards = servers.length > 0
    ? `<div class="flex flex-col gap-1">
        <!-- Header row -->
        <div class="flex items-center gap-2 px-3 pb-1 text-[10px] text-gray-700 uppercase tracking-wider">
          <span class="w-2 shrink-0"></span>
          <span class="w-16 shrink-0">Port</span>
          <span class="w-20 shrink-0">Name</span>
          <span class="flex-1">Command</span>
          <span class="w-12 text-right shrink-0">Uptime</span>
          <span class="w-20 shrink-0"></span>
        </div>
        ${servers.map(s => renderServerListItem(s, proj.id)).join('')}
        <div class="mt-1">${renderAddServerCard(proj.id)}</div>
      </div>`
    : `<div class="flex flex-col items-start gap-2">
        <p class="text-xs text-gray-600">No servers registered.</p>
        ${renderAddServerCard(proj.id)}
      </div>`

  return `<div>${header}${cards}</div>`
}

// ── Event wiring ──────────────────────────────────────────────────────────────

function wireEvents() {
  if (!_el) return

  const searchEl = _el.querySelector('#sl-search')
  if (searchEl) {
    searchEl.addEventListener('input', e => {
      _searchQuery = e.target.value
      render()
    })
    if (document.activeElement === document.body) searchEl.focus()
  }

  const logHeader = _el.querySelector('#log-strip-header')
  if (logHeader) {
    logHeader.addEventListener('click', e => {
      // Let Clear / Export buttons reach the delegated handler instead of toggling.
      if (e.target.closest('[data-action]')) return
      _logExpanded = !_logExpanded
      if (_logExpanded) _logNewCount = 0  // clear indicator when opening (item 5)
      render()
    })
  }
}

async function handleAction(e) {
  const btn = e.target.closest('[data-action]')
  if (!btn) return
  e.stopPropagation()

  const action    = btn.dataset.action
  const serverID  = btn.dataset.serverId
  const projectID = btn.dataset.projectId

  switch (action) {

    case 'select-server':
      if (serverID) {
        setSelectedServer(serverID)
        if (!_logExpanded) {
          _logExpanded = true
          _logPendingScroll = true
        }
        if (getLogLines(serverID).length === 0) {
          try {
            const lines = await GetRecentLogs(serverID, 500)
            if (lines?.length) {
              appendLogLines(serverID, lines)
              _logPendingScroll = true
            }
          } catch (_) {}
        }
        render()
      }
      break

    case 'open-card-menu':
      _openCardMenu(btn, serverID, projectID)
      break

    case 'open-browser': {
      const port = btn.dataset.port
      if (port) BrowserOpenURL(`http://localhost:${port}`)
      break
    }

    case 'suggest-port': {
      const curPort = parseInt(btn.dataset.port, 10)
      if (!serverID || !curPort) break
      try {
        const suggested = await SuggestFreePort(curPort)
        const result = await showModal({
          title: 'Change Port',
          message: `Port :${curPort} is occupied. Pick a free port:`,
          formContent: `
            <input id="new-port" type="number" value="${suggested}" min="1024" max="65535"
              class="w-full mt-2 bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100">
          `,
          actions: [
            { label: 'Cancel',      value: 'cancel', style: 'btn-ghost' },
            { label: 'Change Port', value: 'change', style: 'btn-primary' },
          ],
        })
        if (result !== 'change') break
        const newPort = parseInt(document.getElementById('new-port')?.value, 10)
        if (!newPort || newPort < 1024 || newPort > 65535) {
          await showError('Bad port', 'Port must be 1024–65535.')
          break
        }
        const info = findProjectByServer(serverID)
        if (!info) break
        await UpdateServer(info.project.id, { ...info.server, port: newPort })
        await refreshProjects()
      } catch (err) {
        await showError('Change port failed', err?.message)
      }
      break
    }

    case 'add-project':
      document.querySelector('[data-nav="add-project"]')?.click()
      break

    case 'toggle-autostart':
      if (serverID) {
        const info = findProjectByServer(serverID)
        if (info) {
          const toggled = !info.server.autostart
          await UpdateServer(info.project.id, { ...info.server, autostart: toggled })
          await refreshProjects()
        }
      }
      break

    case 'toggle-autorestart':
      if (serverID) {
        const info = findProjectByServer(serverID)
        if (info) {
          const toggled = !info.server.autorestart
          await UpdateServer(info.project.id, { ...info.server, autorestart: toggled })
          await refreshProjects()
        }
      }
      break

    case 'copy-path': {
      const path = btn.dataset.path
      if (path) {
        navigator.clipboard.writeText(path).then(() => {
          btn.textContent = '✓ Copied!'
          btn.classList.add('bg-green-900/40', 'text-green-300')
          setTimeout(() => {
            btn.textContent = path
            btn.classList.remove('bg-green-900/40', 'text-green-300')
          }, 2000)
        })
      }
      break
    }

    case 'start':
      if (serverID) try { await StartServer(serverID) } catch (err) { await showError('Start failed', err?.message) }
      break

    case 'stop':
      if (serverID) try { await StopServer(serverID) } catch (err) { await showError('Stop failed', err?.message) }
      break

    case 'restart':
      if (serverID) try { await RestartServer(serverID) } catch (err) { await showError('Restart failed', err?.message) }
      break

    case 'restart-all':
      if (projectID) await restartAll(projectID)
      break

    case 'add-server':
      if (projectID) await addServerFlow(projectID)
      break

    case 'resync-port':
      if (serverID) await resyncPortFlow(serverID)
      break

    case 'edit-server':
      if (serverID && projectID) await editServerFlow(serverID, projectID)
      break

    case 'remove-server':
      if (serverID && projectID) await removeServerFlow(serverID, projectID)
      break

    case 'remove-project':
      if (projectID) await removeProjectFlow(projectID)
      break

    case 'goto-project':
      if (projectID) setProjectFilter(projectID)
      break

    case 'clear-log':
      if (serverID) { clearLogLines(serverID); render() }
      break

    case 'export-log':
      if (serverID) await exportLogFlow(serverID)
      break
  }
}

// ── ⋮ Card menu ───────────────────────────────────────────────────────────────

function _openCardMenu(triggerBtn, serverID, projectID) {
  document.querySelector('.card-menu-dropdown')?.remove()

  const info = findProjectByServer(serverID)
  const srv = info?.server
  if (!srv) return

  const autostart = srv.autostart ?? false
  const autorestart = srv.autorestart ?? false

  const rect = triggerBtn.getBoundingClientRect()
  const menu = document.createElement('div')
  menu.className = 'card-menu-dropdown fixed z-50 bg-gray-800 border border-gray-700 rounded-lg shadow-xl py-1 min-w-[160px]'
  menu.style.top = `${rect.bottom + 4}px`
  menu.style.left = `${rect.right - 160}px`
  menu.innerHTML = `
    <button data-action="toggle-autostart" data-server-id="${serverID}"
      class="w-full text-left px-3 py-2 text-xs text-gray-200 hover:bg-gray-700 transition-colors flex items-center gap-2">
      <span class="${autostart ? 'text-green-400' : 'text-gray-600'}">●</span> Autostart
    </button>
    <button data-action="toggle-autorestart" data-server-id="${serverID}"
      class="w-full text-left px-3 py-2 text-xs text-gray-200 hover:bg-gray-700 transition-colors flex items-center gap-2">
      <span class="${autorestart ? 'text-green-400' : 'text-gray-600'}">●</span> Auto-restart
    </button>
    <div class="border-t border-gray-700 my-1"></div>
    <button data-action="resync-port"
      data-server-id="${serverID}" data-project-id="${projectID}"
      class="w-full text-left px-3 py-2 text-xs text-gray-200 hover:bg-gray-700 transition-colors">
      Re-detect port
    </button>
    <button data-action="edit-server"
      data-server-id="${serverID}" data-project-id="${projectID}"
      class="w-full text-left px-3 py-2 text-xs text-gray-200 hover:bg-gray-700 transition-colors">
      Edit server
    </button>
    <button data-action="remove-server"
      data-server-id="${serverID}" data-project-id="${projectID}"
      class="w-full text-left px-3 py-2 text-xs text-red-400 hover:bg-gray-700 transition-colors">
      Remove
    </button>
  `

  document.body.appendChild(menu)
  const close = ev => {
    if (!menu.contains(ev.target) && ev.target !== triggerBtn) {
      menu.remove()
      document.removeEventListener('click', close, true)
    }
  }
  setTimeout(() => document.addEventListener('click', close, true), 0)
}

// ── Log batch handler ─────────────────────────────────────────────────────────

function onLogBatch(e) {
  const { serverID, lines } = e.detail ?? {}
  if (!serverID || !lines?.length) return
  if (getSelectedServerID() === serverID && _logExpanded) {
    appendLogLinesDOM(_el, lines)
  } else {
    // Strip is closed or a different server is selected — show new-line indicator (item 5)
    _logNewCount += lines.length
    const el = document.getElementById('log-new-count')
    if (el) {
      el.textContent = `●${_logNewCount} new`
      el.className = 'text-xs text-indigo-300 bg-indigo-900/50 rounded px-1.5 py-0.5 shrink-0'
    }
  }
}

// ── CRUD flows ────────────────────────────────────────────────────────────────

async function addServerFlow(projectID) {
  const modalPromise = showModal({
    title: 'Add Server',
    message: '',
    formContent: `
      <div class="space-y-2 mt-2">
        <input id="asrv-name" type="text" placeholder="Name (e.g. dev)"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100 placeholder-gray-600">
        <input id="asrv-cmd" type="text" placeholder="Command (e.g. npm run dev)"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100 placeholder-gray-600">
        <div class="flex gap-1.5">
          <input id="asrv-port" type="number" placeholder="Port (e.g. 3000)" min="1024" max="65535"
            class="flex-1 bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100 placeholder-gray-600">
          <button id="asrv-detect" type="button"
            class="shrink-0 px-2.5 bg-gray-700 hover:bg-gray-600 text-gray-300 border border-gray-600
                   rounded text-xs font-medium transition-colors whitespace-nowrap">
            Detect
          </button>
        </div>
        <div id="asrv-hint" class="text-xs text-gray-600 hidden"></div>
        <div id="asrv-live-ports" class="mt-1">
          <p class="text-xs text-gray-500 mb-1">Live ports (click to fill):</p>
          <div id="asrv-port-chips" class="flex flex-wrap gap-1">
            <span class="text-[10px] text-gray-600">Scanning…</span>
          </div>
        </div>
        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
          <input type="checkbox" id="asrv-auto" class="accent-indigo-500">
          Start on VPM launch
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
          <input type="checkbox" id="asrv-autorestart" class="accent-indigo-500">
          Auto-restart on crash
        </label>
      </div>
    `,
    actions: [
      { label: 'Cancel',     value: 'cancel', style: 'btn-ghost' },
      { label: 'Add Server', value: 'add',    style: 'btn-primary' },
    ],
  })

  // Wire live ports chips (async, don't block modal open)
  ;(async () => {
    const chipsEl = document.getElementById('asrv-port-chips')
    if (!chipsEl) return
    try {
      const livePorts = await Promise.race([
        GetListeningPorts(),
        new Promise(r => setTimeout(() => r([]), 3000)),
      ])
      const proj     = getProjects().find(p => p.id === projectID)
      const regPorts = new Set((proj?.servers ?? []).map(s => s.port))
      const unregistered = livePorts.filter(p => !regPorts.has(p) && p >= 1024 && p <= 49151).slice(0, 10)

      if (unregistered.length === 0) {
        chipsEl.innerHTML = '<span class="text-[10px] text-gray-600">None active</span>'
      } else {
        chipsEl.innerHTML = unregistered.map(port => `
          <button type="button" data-port="${port}"
            class="text-[10px] px-2 py-0.5 rounded bg-indigo-900/40 text-indigo-300 hover:bg-indigo-700 transition-colors mono">
            :${port}
          </button>
        `).join('')
        chipsEl.querySelectorAll('[data-port]').forEach(chip => {
          chip.addEventListener('click', e => {
            e.preventDefault()
            const port = chip.dataset.port
            const portInput = document.getElementById('asrv-port')
            if (portInput) portInput.value = port
          })
        })
      }
    } catch (_) {
      chipsEl.innerHTML = '<span class="text-[10px] text-gray-600">Scan failed</span>'
    }
  })()

  // Wire Detect button (modal DOM is synchronously rendered above)
  document.getElementById('asrv-detect')?.addEventListener('click', async () => {
    const btn  = document.getElementById('asrv-detect')
    const hint = document.getElementById('asrv-hint')
    if (btn) { btn.textContent = '…'; btn.disabled = true }

    try {
      const livePorts = await Promise.race([
        GetListeningPorts(),
        new Promise(r => setTimeout(() => r([]), 3000)),
      ])
      const proj      = getProjects().find(p => p.id === projectID)
      const regPorts  = new Set((proj?.servers ?? []).map(s => s.port))
      const candidates = livePorts.filter(p => !regPorts.has(p) && p >= 1024 && p <= 49151)

      if (candidates.length === 0) {
        if (hint) { hint.textContent = 'No unregistered active ports found.'; hint.classList.remove('hidden') }
      } else {
        const portInput = document.getElementById('asrv-port')
        if (portInput) portInput.value = candidates[0]
        if (hint) {
          hint.textContent = candidates.length > 1
            ? `Also active: ${candidates.slice(1, 6).map(p => ':' + p).join('  ')}`
            : ''
          hint.classList.toggle('hidden', candidates.length <= 1)
        }
      }
    } catch (_) {
      if (hint) { hint.textContent = 'Scan timed out.'; hint.classList.remove('hidden') }
    } finally {
      if (btn) { btn.textContent = 'Detect'; btn.disabled = false }
    }
  })

  const result = await modalPromise
  if (result !== 'add') return

  const name        = document.getElementById('asrv-name')?.value?.trim()
  const command     = document.getElementById('asrv-cmd')?.value?.trim()
  const port        = parseInt(document.getElementById('asrv-port')?.value, 10)
  const autostart   = document.getElementById('asrv-auto')?.checked ?? false
  const autorestart = document.getElementById('asrv-autorestart')?.checked ?? false

  if (!name || !command || !port) { await showError('Incomplete', 'Name, command, and port are required.'); return }
  if (port < 1024 || port > 65535) { await showError('Bad port', 'Port must be 1024–65535.'); return }

  try {
    await AddServer(projectID, { id: '', name, command, port, autostart, autorestart })
    await refreshProjects()
  } catch (e) {
    await showError('Add Server failed', e?.message)
  }
}

async function editServerFlow(serverID, projectID) {
  const proj = getProjects().find(p => p.id === projectID)
  const srv  = proj?.servers?.find(s => s.id === serverID)
  if (!srv) return

  const result = await showModal({
    title: 'Edit Server',
    message: '',
    formContent: `
      <div class="space-y-2 mt-2">
        <input id="esrv-name" type="text" value="${esc(srv.name)}" placeholder="Name"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100 placeholder-gray-600">
        <input id="esrv-cmd" type="text" value="${esc(srv.command)}" placeholder="Command"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100 placeholder-gray-600">
        <input id="esrv-port" type="number" value="${srv.port}" min="1024" max="65535"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100">
        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
          <input type="checkbox" id="esrv-auto" ${srv.autostart ? 'checked' : ''} class="accent-indigo-500">
          Start on VPM launch
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
          <input type="checkbox" id="esrv-autorestart" ${srv.autorestart ? 'checked' : ''} class="accent-indigo-500">
          Auto-restart on crash
        </label>
      </div>
    `,
    actions: [
      { label: 'Cancel', value: 'cancel', style: 'btn-ghost' },
      { label: 'Save',   value: 'save',   style: 'btn-primary' },
    ],
  })
  if (result !== 'save') return

  const name        = document.getElementById('esrv-name')?.value?.trim()
  const command     = document.getElementById('esrv-cmd')?.value?.trim()
  const port        = parseInt(document.getElementById('esrv-port')?.value, 10)
  const autostart   = document.getElementById('esrv-auto')?.checked ?? false
  const autorestart = document.getElementById('esrv-autorestart')?.checked ?? false

  if (!name || !command || !port) { await showError('Incomplete', 'Name, command, and port are required.'); return }

  try {
    await UpdateServer(projectID, { id: serverID, name, command, port, autostart, autorestart })
    await refreshProjects()
  } catch (e) {
    await showError('Update failed', e?.message)
  }
}

async function removeServerFlow(serverID, projectID) {
  const info  = findProjectByServer(serverID)
  const label = info ? `${info.server.name} (:${info.server.port})` : serverID
  const ok = await showConfirm('Remove Server', `Remove "${label}"? This cannot be undone.`)
  if (!ok) return
  try {
    await RemoveServer(projectID, serverID)
    if (getSelectedServerID() === serverID) setSelectedServer(null)
    await refreshProjects()
  } catch (e) {
    await showError('Remove failed', e?.message)
  }
}

async function removeProjectFlow(projectID) {
  const proj = getProjects().find(p => p.id === projectID)
  const ok = await showConfirm('Remove Project',
    `Remove project "${proj?.name ?? projectID}" and all its servers?`)
  if (!ok) return
  try {
    await RemoveProject(projectID)
    setProjectFilter(null)
    await refreshProjects()
  } catch (e) {
    await showError('Remove failed', e?.message)
  }
}

async function restartAll(projectID) {
  const proj = getProjects().find(p => p.id === projectID)
  if (!proj) return
  for (const s of (proj.servers ?? [])) {
    const st = getServerState(s.id)
    try {
      if (st === 'RUNNING') {
        await RestartServer(s.id)
      } else if (st === 'STOPPED' || st === 'ERROR') {
        await StartServer(s.id)
      }
    } catch (_) {}
  }
}

async function exportLogFlow(serverID) {
  const info   = findProjectByServer(serverID)
  const projID = info?.project?.id
  if (!projID) return

  let dest
  try {
    dest = await BrowseDirectory()   // native folder picker
  } catch (e) {
    await showError('Export failed', e?.message)
    return
  }
  if (!dest) return  // user cancelled

  try {
    await ExportLogs(projID, dest)
    await showModal({ title: 'Exported', message: `Logs copied to: ${dest}` })
  } catch (e) {
    await showError('Export failed', e?.message)
  }
}

// resyncPortFlow: re-detect a VPM-spawned server's actual port (port drift).
async function resyncPortFlow(serverID) {
  const before = findProjectByServer(serverID)?.server?.port
  try {
    const port = await ResyncServerPort(serverID)
    await refreshProjects()
    const msg = (before && port === before)
      ? `Still on port :${port} — no change.`
      : `Re-detected — now tracking port :${port}.`
    await showModal({ title: 'Re-detect port', message: msg })
  } catch (e) {
    await showError('Re-detect failed', e?.message)
  }
}

// applySuggestedPort: collision alert "Use free port" → reassign + relaunch.
async function applySuggestedPort({ serverID, port } = {}) {
  if (!serverID || !port) return
  const info = findProjectByServer(serverID)
  if (!info) return
  try {
    await UpdateServer(info.project.id, { ...info.server, port })
    await refreshProjects()
    await StartServer(serverID)   // collision left it stopped; launch on the free port
  } catch (e) {
    await showError('Could not switch port', e?.message)
  }
}

// ── Shared helpers ────────────────────────────────────────────────────────────

async function refreshProjects() {
  const projects = await ListProjects()
  setProjects(projects)
}

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}
