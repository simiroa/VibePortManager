// components/system-ports.js — System Port Analyzer + Port Killer + Registration
// Single-modal inline router: list → detail → add / new, all rendered inside
// #sp-body so the scan window never closes when an item is clicked.
// Unregistered ports can be added to existing projects or create new ones, with
// the start command auto-detected from the listening process (blank = monitor-only).

import { ScanSystemPorts, KillByPort, AddProject, AddServer, ListProjects, GetProcessCommand } from '../wails.js'
import { showModal } from '../modal.js'
import { getProjects, findProjectByPort } from '../store.js'
import { setProjectFilter } from '../state.js'

let _cachedEntries = null

export async function openSystemPorts() {
  const promise = showModal({
    title: 'System Ports',
    message: 'All listening ports across Windows and WSL backends.',
    formContent: `<div id="sp-body" class="mt-2"><p class="text-xs text-gray-500">Scanning…</p></div>`,
    actions: [{ label: 'Close', value: 'close', style: 'btn-ghost' }],
  })
  refresh()
  await promise
}

function body() {
  return document.getElementById('sp-body')
}

// ── List view ───────────────────────────────────────────────────────────────

async function refresh() {
  const el = body()
  if (!el) return

  if (_cachedEntries !== null) {
    renderEntries(_cachedEntries, el)
    return
  }

  el.innerHTML = `<p class="text-xs text-gray-500">Scanning…</p>`
  try {
    _cachedEntries = (await ScanSystemPorts()) ?? []
  } catch (e) {
    el.innerHTML = `<p class="text-xs text-red-400">Scan failed: ${esc(e?.message)}</p>`
    return
  }
  renderEntries(_cachedEntries, el)
}

function renderEntries(entries, el) {
  entries.sort((a, b) => a.port - b.port)
  const managed = new Set()
  for (const p of getProjects()) for (const s of (p.servers ?? [])) managed.add(s.port)

  el.innerHTML = `
    <div class="flex items-center justify-between mb-2">
      <span class="text-xs text-gray-500">${entries.length} listening port${entries.length !== 1 ? 's' : ''}</span>
      <button id="sp-rescan" class="btn-xs btn-gray">Rescan</button>
    </div>
    <div class="max-h-80 overflow-y-auto border border-gray-800 rounded divide-y divide-gray-800">
      ${entries.length === 0
        ? `<p class="px-3 py-4 text-xs text-gray-600 text-center">No listening ports found.</p>`
        : entries.map(rowHTML(managed)).join('')}
    </div>
  `
  document.getElementById('sp-rescan')?.addEventListener('click', () => {
    _cachedEntries = null
    refresh()
  })
  el.querySelectorAll('[data-port-info]').forEach(row =>
    row.addEventListener('click', e => {
      if (e.target.closest('[data-kill-port], [data-confirm-kill], [data-cancel-kill]')) return
      showDetail(JSON.parse(row.dataset.portInfo))
    }))
  el.querySelectorAll('[data-kill-port]').forEach(btn =>
    btn.addEventListener('click', () => onKillClick(btn)))
}

const rowHTML = managed => e => `
  <div class="flex items-center gap-2 px-3 py-1.5 text-xs cursor-pointer hover:bg-gray-800/50 transition-colors"
       data-port-info='${JSON.stringify({port: e.port, processName: e.processName, pid: e.pid, backendId: e.backendId})}'>
    <span class="mono text-indigo-300 w-14 shrink-0 font-semibold">:${e.port}</span>
    <span class="text-gray-300 flex-1 truncate">
      ${esc(e.processName || '?')} <span class="text-gray-600">pid ${e.pid}</span>
    </span>
    <span class="text-gray-600 shrink-0 text-[10px]">${esc(shortBackend(e.backendId))}</span>
    ${managed.has(e.port) ? `<span class="text-amber-400 shrink-0 text-[10px]" title="Managed by VPM">managed</span>` : ''}
    <span class="kill-slot shrink-0 flex items-center gap-1">
      <button data-kill-port="${e.port}"
        class="text-red-400 hover:text-red-300 px-1.5 py-0.5 rounded hover:bg-red-900/30 transition-colors">Kill</button>
    </span>
  </div>
`

// ── Detail view (inline — keeps the scan window open) ─────────────────────────

function showDetail(info) {
  const el = body()
  if (!el) return
  const { port, processName, pid, backendId } = info

  const managed = findProjectByPort(port)
  const managerLabel = managed
    ? `${esc(managed.server.name)} <span class="text-gray-500">(${esc(managed.project.name)})</span>`
    : `<span class="text-gray-500">Not managed by VPM</span>`

  el.innerHTML = `
    <button id="sp-back" class="text-xs text-gray-400 hover:text-gray-200 mb-3 transition-colors">← Back to list</button>
    <div class="rounded-lg border border-gray-800 bg-gray-900/40 p-4 space-y-3">
      <div class="flex items-baseline gap-2">
        <span class="mono text-lg font-bold text-indigo-300">:${port}</span>
        <span class="text-xs text-gray-600">${esc(shortBackend(backendId))}</span>
      </div>
      ${detailRow('Process', `${esc(processName || '?')} <span class="text-gray-500">(PID ${pid})</span>`)}
      ${detailRow('VPM status', managerLabel)}
    </div>
    <div class="flex flex-wrap gap-2 justify-end mt-4">
      ${managed
        ? `<button id="sp-goto" class="btn-primary">Go to Project</button>`
        : `<button id="sp-add" class="btn-primary">Add to Project…</button>
           <button id="sp-new" class="btn-ghost">New Project…</button>`}
      <button id="sp-kill" class="btn-red">Kill process</button>
    </div>
  `

  document.getElementById('sp-back')?.addEventListener('click', () => refresh())
  document.getElementById('sp-goto')?.addEventListener('click', () => {
    if (managed) {
      setProjectFilter(managed.project.id)
      document.querySelector('[data-nav="system-ports"]')?.click()
    }
  })
  document.getElementById('sp-add')?.addEventListener('click', () => renderAddForm(info))
  document.getElementById('sp-new')?.addEventListener('click', () => renderNewForm(info))
  document.getElementById('sp-kill')?.addEventListener('click', async ev => {
    const btn = ev.currentTarget
    btn.textContent = 'killing…'
    btn.disabled = true
    try {
      await KillByPort(port)
    } catch (e) {
      btn.textContent = 'failed'
      return
    }
    _cachedEntries = null
    refresh()
  })
}

function detailRow(label, valueHTML) {
  return `
    <div>
      <p class="text-[11px] uppercase tracking-wide text-gray-600">${label}</p>
      <p class="text-sm text-gray-100">${valueHTML}</p>
    </div>
  `
}

// ── Add to existing project (inline form) ─────────────────────────────────────

function renderAddForm(info) {
  const el = body()
  if (!el) return
  const { port, processName, pid, backendId } = info
  const projects = getProjects().filter(p => p.kind !== 'service')
  const projectOptions = projects.map(p => `<option value="${esc(p.id)}">${esc(p.name)}</option>`).join('')

  el.innerHTML = `
    <button id="sp-back" class="text-xs text-gray-400 hover:text-gray-200 mb-3 transition-colors">← Back</button>
    <h3 class="text-sm font-semibold text-gray-200 mb-2">Add :${port} to a project</h3>
    <div class="space-y-2">
      <select id="sp-project-select" class="input">${projectOptions}</select>
      <div>
        <input id="sp-server-name" type="text" placeholder="Server name (e.g. dev)" value="${esc(cleanName(processName))}" class="input">
        <div id="sp-name-err" class="text-xs text-red-400 mt-0.5 hidden"></div>
      </div>
      ${cmdField()}
    </div>
    <div class="flex gap-2 justify-end mt-4">
      <button id="sp-cancel" class="btn-ghost">Cancel</button>
      <button id="sp-register" class="btn-primary">Register</button>
    </div>
  `

  wireCmdField(pid, backendId)
  document.getElementById('sp-back')?.addEventListener('click', () => showDetail(info))
  document.getElementById('sp-cancel')?.addEventListener('click', () => showDetail(info))
  document.getElementById('sp-register')?.addEventListener('click', async () => {
    const projectId = document.getElementById('sp-project-select')?.value
    const name = document.getElementById('sp-server-name')?.value?.trim()
    const command = document.getElementById('sp-server-cmd')?.value?.trim()
    if (!fieldRequired('sp-server-name', 'sp-name-err', name, 'Server name is required.')) return
    if (!projectId) return

    try {
      await AddServer(projectId, { id: '', name, command: command || '', port, autostart: false, autorestart: false })
      await refreshProjects()
      _cachedEntries = null
      await refresh()
      toast(`":${port}" registered${command ? '' : ' (monitor-only)'}.`)
    } catch (e) {
      showFieldErr('sp-name-err', e?.message || 'Registration failed')
    }
  })
}

// ── New project (inline form) ─────────────────────────────────────────────────

function renderNewForm(info) {
  const el = body()
  if (!el) return
  const { port, processName, pid, backendId } = info

  el.innerHTML = `
    <button id="sp-back" class="text-xs text-gray-400 hover:text-gray-200 mb-3 transition-colors">← Back</button>
    <h3 class="text-sm font-semibold text-gray-200 mb-2">New project for :${port}</h3>
    <div class="space-y-2">
      <div>
        <input id="sp-proj-path" type="text" placeholder="Project path (e.g. C:\\projects\\myapp)" class="input">
        <div id="sp-path-err" class="text-xs text-red-400 mt-0.5 hidden"></div>
      </div>
      <div>
        <input id="sp-proj-name" type="text" placeholder="Project name" class="input">
        <div id="sp-pname-err" class="text-xs text-red-400 mt-0.5 hidden"></div>
      </div>
      <div>
        <input id="sp-server-name" type="text" placeholder="Server name" value="${esc(cleanName(processName))}" class="input">
        <div id="sp-name-err" class="text-xs text-red-400 mt-0.5 hidden"></div>
      </div>
      ${cmdField()}
    </div>
    <div class="flex gap-2 justify-end mt-4">
      <button id="sp-cancel" class="btn-ghost">Cancel</button>
      <button id="sp-create" class="btn-primary">Create</button>
    </div>
  `

  wireCmdField(pid, backendId)
  document.getElementById('sp-back')?.addEventListener('click', () => showDetail(info))
  document.getElementById('sp-cancel')?.addEventListener('click', () => showDetail(info))
  document.getElementById('sp-create')?.addEventListener('click', async () => {
    const path = document.getElementById('sp-proj-path')?.value?.trim()
    const projName = document.getElementById('sp-proj-name')?.value?.trim()
    const srvName = document.getElementById('sp-server-name')?.value?.trim()
    const command = document.getElementById('sp-server-cmd')?.value?.trim()

    let ok = true
    ok = fieldRequired('sp-proj-path', 'sp-path-err', path, 'Project path is required.') && ok
    ok = fieldRequired('sp-proj-name', 'sp-pname-err', projName, 'Project name is required.') && ok
    ok = fieldRequired('sp-server-name', 'sp-name-err', srvName, 'Server name is required.') && ok
    if (!ok) return

    try {
      const proj = await AddProject(path, projName, null)
      await AddServer(proj.id, { id: '', name: srvName, command: command || '', port, autostart: false, autorestart: false })
      await refreshProjects()
      setProjectFilter(proj.id)
      _cachedEntries = null
      await refresh()
      toast(`Project "${esc(projName)}" created on :${port}${command ? '' : ' (monitor-only)'}.`)
    } catch (e) {
      showFieldErr('sp-path-err', e?.message || 'Creation failed')
    }
  })
}

// ── Command field (auto-detect from the listening process) ────────────────────

function cmdField() {
  return `
    <div>
      <div class="flex gap-1.5">
        <input id="sp-server-cmd" type="text" placeholder="Command — blank to monitor only" class="input mono flex-1">
        <button id="sp-detect" type="button" class="btn-xs btn-gray shrink-0 whitespace-nowrap" title="Detect from running process">🔍 Detect</button>
      </div>
      <p class="text-[11px] text-gray-600 mt-0.5">Leave blank to track the port without a start command.</p>
    </div>
  `
}

function wireCmdField(pid, backendId) {
  const detect = async () => {
    const input = document.getElementById('sp-server-cmd')
    const btn = document.getElementById('sp-detect')
    if (!input) return
    const prev = btn ? btn.textContent : ''
    if (btn) { btn.textContent = '…'; btn.disabled = true }
    try {
      const cmd = await GetProcessCommand(pid, backendId)
      if (cmd && !input.value.trim()) input.value = cmd
    } catch (_) { /* unknown — leave blank for monitor-only */ }
    if (btn) { btn.textContent = prev; btn.disabled = false }
  }
  document.getElementById('sp-detect')?.addEventListener('click', detect)
  detect() // auto-fill once on open
}

// ── Kill (list row) ───────────────────────────────────────────────────────────

function onKillClick(btn) {
  const slot = btn.closest('.kill-slot')
  const port = btn.dataset.killPort
  if (!slot) return

  slot.innerHTML = `
    <button data-confirm-kill class="text-red-300 bg-red-900/40 hover:bg-red-900/60 px-1.5 py-0.5 rounded transition-colors">Confirm</button>
    <button data-cancel-kill class="text-gray-400 hover:text-gray-200 px-1.5 py-0.5 transition-colors">✕</button>
  `
  slot.querySelector('[data-cancel-kill]')?.addEventListener('click', refresh)
  slot.querySelector('[data-confirm-kill]')?.addEventListener('click', async () => {
    slot.innerHTML = `<span class="text-gray-500">killing…</span>`
    try {
      await KillByPort(parseInt(port, 10))
    } catch (e) {
      slot.innerHTML = `<span class="text-red-400" title="${esc(e?.message)}">failed</span>`
      return
    }
    _cachedEntries = null
    await refresh()
  })
}

// ── Helpers ─────────────────────────────────────────────────────────────────

async function refreshProjects() {
  try {
    const projects = await ListProjects()
    const { setProjects } = await import('../store.js')
    setProjects(projects ?? [])
  } catch (_) {}
}

// fieldRequired shows/clears an inline error and returns whether the value is set.
function fieldRequired(inputId, errId, value, msg) {
  if (!value) { showFieldErr(errId, msg); markInvalid(inputId, true); return false }
  hideFieldErr(errId); markInvalid(inputId, false)
  return true
}

function showFieldErr(errId, msg) {
  const el = document.getElementById(errId)
  if (el) { el.textContent = msg; el.classList.remove('hidden') }
}
function hideFieldErr(errId) {
  const el = document.getElementById(errId)
  if (el) el.classList.add('hidden')
}
function markInvalid(inputId, bad) {
  const el = document.getElementById(inputId)
  if (el) el.classList.toggle('border-red-500', bad)
}

// toast shows a transient confirmation at the bottom of #sp-body.
function toast(msg) {
  const el = body()
  if (!el) return
  const t = document.createElement('div')
  t.className = 'mt-2 text-xs text-green-400 bg-green-900/30 border border-green-800/50 rounded px-2 py-1'
  t.innerHTML = msg
  el.appendChild(t)
  setTimeout(() => t.remove(), 3000)
}

// cleanName turns "node.exe (PID 1234)" / "node.exe" into a tidy default name.
function cleanName(processName) {
  if (!processName) return 'server'
  return processName.replace(/\.exe$/i, '').replace(/\s*\(PID.*\)$/i, '').trim() || 'server'
}

function shortBackend(id) {
  if (!id) return ''
  if (id === 'windows-native') return 'win'
  return id.replace(/^wsl:?/i, 'wsl:')
}

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}
