// components/system-ports.js — System Port Analyzer + Port Killer + Registration
// Lists every listening port, with caching to avoid repeated scans.
// Unregistered ports can be added to existing projects or create new ones.

import { ScanSystemPorts, KillByPort, AddProject, AddServer, ListProjects } from '../wails.js'
import { showModal, showError } from '../modal.js'
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

async function refresh() {
  const body = document.getElementById('sp-body')
  if (!body) return

  if (_cachedEntries !== null) {
    renderEntries(_cachedEntries, body)
    return
  }

  body.innerHTML = `<p class="text-xs text-gray-500">Scanning…</p>`
  try {
    _cachedEntries = (await ScanSystemPorts()) ?? []
  } catch (e) {
    body.innerHTML = `<p class="text-xs text-red-400">Scan failed: ${esc(e?.message)}</p>`
    return
  }

  renderEntries(_cachedEntries, body)
}

function renderEntries(entries, body) {
  entries.sort((a, b) => a.port - b.port)
  const managed = new Set()
  for (const p of getProjects()) for (const s of (p.servers ?? [])) managed.add(s.port)

  body.innerHTML = `
    <div class="flex items-center justify-between mb-2">
      <span class="text-xs text-gray-500">${entries.length} listening port${entries.length !== 1 ? 's' : ''}</span>
      <button id="sp-rescan" class="text-xs px-2 py-1 rounded bg-gray-800 text-gray-300 hover:bg-gray-700 transition-colors">Rescan</button>
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
  body.querySelectorAll('[data-port-info]').forEach(row =>
    row.addEventListener('click', e => {
      if (e.target.closest('[data-kill-port], [data-confirm-kill], [data-cancel-kill]')) return
      onPortRowClick(JSON.parse(row.dataset.portInfo))
    }))
  body.querySelectorAll('[data-kill-port]').forEach(btn =>
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

function onPortRowClick(info) {
  const { port, processName, pid, backendId } = info
  if (!port) return

  const managedServer = findProjectByPort(port)
  const managerLabel = managedServer
    ? `${managedServer.server.name} (${managedServer.project.name})`
    : 'Not managed by VPM'

  const content = `
    <div class="space-y-2 mt-2 text-sm">
      <div>
        <p class="text-xs text-gray-500">Port</p>
        <p class="mono text-indigo-300 font-semibold">:${port}</p>
      </div>
      <div>
        <p class="text-xs text-gray-500">Process</p>
        <p class="text-gray-100">${esc(processName || '?')} (PID ${pid})</p>
      </div>
      <div>
        <p class="text-xs text-gray-500">Backend</p>
        <p class="text-gray-100">${esc(shortBackend(backendId))}</p>
      </div>
      <div class="border-t border-gray-700 pt-2">
        <p class="text-xs text-gray-500">VPM Status</p>
        <p class="text-gray-100">${managerLabel}</p>
      </div>
    </div>
  `

  const actions = managedServer
    ? [
        { label: 'Go to Project', value: 'goto', style: 'btn-primary' },
        { label: 'Close', value: 'close', style: 'btn-ghost' },
      ]
    : [
        { label: 'Add to Project…', value: 'add', style: 'btn-primary' },
        { label: 'New Project…', value: 'new', style: 'btn-ghost' },
        { label: 'Close', value: 'close', style: 'btn-ghost' },
      ]

  showModal({
    title: `Port :${port}`,
    message: '',
    formContent: content,
    actions,
  }).then(async result => {
    if (result === 'goto' && managedServer) {
      setProjectFilter(managedServer.project.id)
      document.querySelector('[data-nav="system-ports"]')?.click()
    } else if (result === 'add') {
      await addToExistingProject(port, processName)
    } else if (result === 'new') {
      await createNewProject(port, processName)
    }
  })
}

async function addToExistingProject(port, processName) {
  const projects = getProjects()
  const projectOptions = projects.map(p => `<option value="${esc(p.id)}">${esc(p.name)}</option>`).join('')

  const result = await showModal({
    title: 'Add to Project',
    message: '',
    formContent: `
      <div class="space-y-2 mt-2">
        <select id="sp-project-select" class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
          ${projectOptions}
        </select>
        <input id="sp-server-name" type="text" placeholder="Server name (e.g. dev)" value="${esc(processName || 'server')}"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
        <input id="sp-server-cmd" type="text" placeholder="Command (e.g. npm run dev)"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100">
      </div>
    `,
    actions: [
      { label: 'Cancel', value: 'cancel', style: 'btn-ghost' },
      { label: 'Register', value: 'register', style: 'btn-primary' },
    ],
  })

  if (result !== 'register') return

  const projectId = document.getElementById('sp-project-select')?.value
  const name = document.getElementById('sp-server-name')?.value?.trim()
  const command = document.getElementById('sp-server-cmd')?.value?.trim()

  if (!projectId || !name || !command) {
    await showError('Incomplete', 'Project, name, and command are required.')
    return
  }

  try {
    await AddServer(projectId, { id: '', name, command, port, autostart: false, autorestart: false })
    await refreshProjects()
    await showModal({ title: 'Registered', message: `Server ":${port}" added to project.` })
  } catch (e) {
    await showError('Registration failed', e?.message)
  }
}

async function createNewProject(port, processName) {
  const result = await showModal({
    title: 'New Project',
    message: 'Register this port as a new project.',
    formContent: `
      <div class="space-y-2 mt-2">
        <input id="sp-proj-path" type="text" placeholder="Project path (e.g. C:\\projects\\myapp)"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
        <input id="sp-proj-name" type="text" placeholder="Project name"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
        <input id="sp-server-name2" type="text" placeholder="Server name" value="${esc(processName || 'server')}"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
        <input id="sp-server-cmd2" type="text" placeholder="Command (e.g. npm run dev)"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100">
      </div>
    `,
    actions: [
      { label: 'Cancel', value: 'cancel', style: 'btn-ghost' },
      { label: 'Create', value: 'create', style: 'btn-primary' },
    ],
  })

  if (result !== 'create') return

  const path = document.getElementById('sp-proj-path')?.value?.trim()
  const projName = document.getElementById('sp-proj-name')?.value?.trim()
  const srvName = document.getElementById('sp-server-name2')?.value?.trim()
  const srvCmd = document.getElementById('sp-server-cmd2')?.value?.trim()

  if (!path || !projName || !srvName || !srvCmd) {
    await showError('Incomplete', 'All fields are required.')
    return
  }

  try {
    const proj = await AddProject(path, projName, null)
    await AddServer(proj.id, { id: '', name: srvName, command: srvCmd, port, autostart: false, autorestart: false })
    await refreshProjects()
    setProjectFilter(proj.id)
    await showModal({ title: 'Created', message: `Project "${projName}" created with server on port :${port}.` })
  } catch (e) {
    await showError('Creation failed', e?.message)
  }
}

async function refreshProjects() {
  try {
    const projects = await ListProjects()
    const { setProjects } = await import('../store.js')
    setProjects(projects ?? [])
  } catch (_) {}
}

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

function shortBackend(id) {
  if (!id) return ''
  if (id === 'windows-native') return 'win'
  return id.replace(/^wsl:?/i, 'wsl:')
}

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}
