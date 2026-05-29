// components/system-ports.js — System Port Analyzer + Port Killer panel
// Lists every listening port across Windows + WSL backends with a Kill action.
// Opened from the sidebar; rendered inside the shared modal overlay.
// Kill uses an inline two-step confirm (no nested modal — the overlay is shared).

import { ScanSystemPorts, KillByPort } from '../wails.js'
import { showModal } from '../modal.js'
import { getProjects, findProjectByServer } from '../store.js'
import { setProjectFilter } from '../state.js'

export async function openSystemPorts() {
  const promise = showModal({
    title: 'System Ports',
    message: 'All listening ports across Windows and WSL backends.',
    formContent: `<div id="sp-body" class="mt-2"><p class="text-xs text-gray-500">Scanning…</p></div>`,
    actions: [{ label: 'Close', value: 'close', style: 'btn-ghost' }],
  })
  // The modal DOM is rendered synchronously by showModal above; populate it now.
  refresh()
  await promise
}

async function refresh() {
  const body = document.getElementById('sp-body')
  if (!body) return // modal was closed
  body.innerHTML = `<p class="text-xs text-gray-500">Scanning…</p>`

  let entries = []
  try {
    entries = (await ScanSystemPorts()) ?? []
  } catch (e) {
    body.innerHTML = `<p class="text-xs text-red-400">Scan failed: ${esc(e?.message)}</p>`
    return
  }
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
  document.getElementById('sp-rescan')?.addEventListener('click', refresh)
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

  // Find if any VPM server is on this port
  const managedServer = Array.from(getProjects())
    .flatMap(p => (p.servers ?? []).map(s => ({ ...s, projectId: p.id })))
    .find(s => s.port === port)

  const managerLabel = managedServer
    ? `${managedServer.name} (${getProjects().find(p => p.id === managedServer.projectId)?.name})`
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
        { label: 'Close', value: 'close', style: 'btn-ghost' },
      ]

  showModal({
    title: `Port :${port}`,
    message: '',
    formContent: content,
    actions,
  }).then(result => {
    if (result === 'goto' && managedServer) {
      setProjectFilter(managedServer.projectId)
      // Close the System Ports modal by simulating close button click
      document.querySelector('[data-nav="system-ports"]')?.click()
    }
  })
}

function onKillClick(btn) {
  const slot = btn.closest('.kill-slot')
  const port = btn.dataset.killPort
  if (!slot) return

  slot.innerHTML = `
    <button data-confirm-kill class="text-red-300 bg-red-900/40 hover:bg-red-900/60 px-1.5 py-0.5 rounded transition-colors">Confirm</button>
    <button data-cancel-kill class="text-gray-400 hover:text-gray-200 px-1.5 py-0.5 transition-colors">✕</button>
  `
  // Cancel → rescan resets the row to its default state.
  slot.querySelector('[data-cancel-kill]')?.addEventListener('click', refresh)
  slot.querySelector('[data-confirm-kill]')?.addEventListener('click', async () => {
    slot.innerHTML = `<span class="text-gray-500">killing…</span>`
    try {
      await KillByPort(parseInt(port, 10))
    } catch (e) {
      slot.innerHTML = `<span class="text-red-400" title="${esc(e?.message)}">failed</span>`
      return
    }
    await refresh()
  })
}

function shortBackend(id) {
  if (!id) return ''
  if (id === 'windows-native') return 'win'
  return id.replace(/^wsl:?/i, 'wsl:') // "wsl:Ubuntu"
}

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}
