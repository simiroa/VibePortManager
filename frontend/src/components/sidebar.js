// components/sidebar.js — sidebar: All | project list | + Add Project
// No tab navigation. View Filter drives server-list rendering.

import { getProjects, getServerState, setProjects } from '../store.js'
import {
  getProjectFilter, setProjectFilter,
  getSelectedServerID,
} from '../state.js'
import {
  AddProject, AddServer, ListProjects,
  AnalyzeProject, GetListeningPorts, BrowseDirectory,
} from '../wails.js'
import { showModal, showError, showLoadingModal, updateLoadingStatus, closeModal } from '../modal.js'

// ── Entry point ───────────────────────────────────────────────────────────────

export function init(sidebarEl) {
  render(sidebarEl)

  window.addEventListener('vpm:projects-updated', () => render(sidebarEl))
  window.addEventListener('vpm:state-changed',    () => render(sidebarEl))
  window.addEventListener('vpm:filter-changed',   () => render(sidebarEl))
}

// ── Render ────────────────────────────────────────────────────────────────────

function render(el) {
  if (!el) return
  const projects = getProjects()
  const filter   = getProjectFilter()

  el.innerHTML = `
    <!-- Top actions -->
    <div class="shrink-0 px-2 pt-2 pb-1 space-y-0.5">
      <button data-nav="add-project"
        class="w-full text-left px-3 py-2 rounded text-sm text-indigo-400
               hover:bg-indigo-900/30 hover:text-indigo-300 transition-colors font-medium">
        + Add Project
      </button>
    </div>

    <!-- All -->
    <div class="px-2 pt-1">
      <button data-nav="all"
        class="w-full text-left px-3 py-2 rounded text-sm transition-colors
          ${filter === null
            ? 'bg-indigo-700 text-white font-medium'
            : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200'}">
        ⚡ All Servers
      </button>
    </div>

    <!-- Project list -->
    <div class="flex-1 overflow-y-auto min-h-0 px-2 mt-1">
      ${projects.length === 0
        ? '<p class="px-3 py-2 text-xs text-gray-600">No projects yet.</p>'
        : projects.map(p => projectRow(p, filter)).join('')}
    </div>

    <!-- Bottom actions -->
    <div class="shrink-0 px-2 pb-2 border-t border-gray-800 pt-2 space-y-0.5">
      <button data-nav="system-ports"
        class="w-full text-left px-3 py-2 rounded text-sm text-gray-400
               hover:bg-gray-800 hover:text-gray-200 transition-colors">
        🔌 System Ports
      </button>
    </div>
  `

  // Wire clicks
  el.querySelector('[data-nav="all"]')?.addEventListener('click', () => setProjectFilter(null))
  el.querySelector('[data-nav="add-project"]')?.addEventListener('click', () => runAddProjectFlow())
  el.querySelector('[data-nav="system-ports"]')?.addEventListener('click', () => {
    import('./system-ports.js').then(m => m.openSystemPorts())
  })

  el.querySelectorAll('[data-project-id]').forEach(btn => {
    btn.addEventListener('click', () => {
      const id = btn.dataset.projectId
      setProjectFilter(filter === id ? null : id)
    })
  })
}

function projectRow(p, filter) {
  const servers  = p.servers ?? []
  const color    = _projectStatusColor(servers)
  const active   = filter === p.id
  const running  = servers.filter(s => getServerState(s.id) === 'RUNNING').length
  const chip     = running > 0 ? `${running}/${servers.length}` : `${servers.length}`
  const chipColor = running > 0 ? 'text-green-400' : 'text-gray-600'

  return `
    <button data-project-id="${esc(p.id)}"
      class="w-full text-left rounded-r py-1.5 mb-0.5 text-xs transition-colors select-none
        ${active ? 'bg-gray-800 text-gray-100' : 'text-gray-400 hover:bg-gray-800/60 hover:text-gray-300'}"
      style="border-left: 3px solid ${color}; padding-left: 9px; padding-right: 8px;">
      <div class="flex items-center gap-1.5">
        <span class="truncate flex-1 font-medium">${esc(p.name)}</span>
        <span class="mono shrink-0 text-xs ${chipColor}">${chip}</span>
      </div>
    </button>
  `
}

function _projectStatusColor(servers) {
  const states = servers.map(s => getServerState(s.id))
  if (states.some(st => st === 'ERROR' || st === 'PORT_COLLISION')) return '#ef4444'
  if (states.some(st => st === 'RUNNING'))                          return '#22c55e'
  if (states.some(st => st === 'STARTING' || st === 'STOPPING'))   return '#facc15'
  return '#374151'
}

// ── Detection flow ────────────────────────────────────────────────────────────
// Two-phase: static analysis first, Detection Mode if that fails.

async function runAddProjectFlow() {
  // ── Step 1: path + name ───────────────────────────────────────────────────
  const step1Promise = showModal({
    title: 'Add Project',
    message: 'Enter the project folder path.',
    formContent: `
      <div class="space-y-2 mt-2">
        <div class="flex gap-1.5">
          <input id="det-path" type="text"
            placeholder="C:\\Users\\me\\my-app"
            class="flex-1 bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm
                   text-gray-100 placeholder-gray-600 focus:outline-none focus:border-indigo-500">
          <button id="det-browse" type="button"
            title="Browse for folder"
            class="shrink-0 px-2.5 bg-gray-700 hover:bg-gray-600 text-gray-300
                   border border-gray-600 rounded transition-colors text-sm">⊞</button>
        </div>
        <input id="det-name" type="text" placeholder="Display name (optional)"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm
                 text-gray-100 placeholder-gray-600 focus:outline-none focus:border-indigo-500">
      </div>
    `,
    actions: [
      { label: 'Cancel',      value: 'cancel', style: 'btn-ghost' },
      { label: 'Continue →',  value: 'next',   style: 'btn-primary' },
    ],
  })
  // Wire browse button (modal DOM is already rendered synchronously above)
  document.getElementById('det-browse')?.addEventListener('click', async () => {
    try {
      const dir = await BrowseDirectory()
      if (dir) {
        const inp = document.getElementById('det-path')
        if (inp) inp.value = dir
      }
    } catch (_) {}
  })
  const s1 = await step1Promise
  if (s1 !== 'next') return

  // Read DOM values synchronously before any async work
  const path = document.getElementById('det-path')?.value?.trim()
  if (!path) return
  const rawName     = document.getElementById('det-name')?.value?.trim()
  const displayName = rawName || path.split(/[/\\]+/).filter(Boolean).pop() || 'Project'

  // ── Step 2a: static analysis only (file reads — fast, no netstat) ────────
  showLoadingModal('Analyzing Project', path)
  let analysis
  try {
    analysis = await AnalyzeProject(path)
  } catch (e) {
    closeModal()
    await showError('Analysis failed', e?.message ?? 'Could not read the project folder.')
    return
  }
  closeModal()

  // ── Multi-server import (workspaces / PM2 / docker-compose) ───────────────
  // Use the import flow when a structured multi-service source was found, or
  // whenever there are 2+ candidates. A plain single app falls through to the
  // single-server proposal + Detection Mode below.
  const detected = Array.isArray(analysis?.servers) ? analysis.servers : []
  const structured = detected.some(s => s.source && s.source !== 'package')
  if (detected.length >= 2 || (detected.length >= 1 && structured)) {
    await runMultiServerImport(path, displayName, detected)
    return
  }

  let detectedPort = analysis?.port ?? null
  const suggestedCmd = analysis?.command ?? ''
  const suggestedScript = analysis?.scriptName ?? 'dev'

  // ── Step 2b: Detection Mode (if static analysis found no port) ─────────────
  // detectedPorts is always int[] after this block.
  let detectedPorts = detectedPort != null ? [detectedPort] : null
  if (detectedPorts === null) {
    // Warn if analysis found nothing — likely an invalid path or non-JS project.
    if (!suggestedCmd && (!analysis?.packageMgr || analysis.packageMgr === 'none')) {
      const proceed = await showModal({
        title: 'Project Not Recognised',
        message: `No package.json or known config found at:\n${path}\n\nThe path may be wrong, or this isn't a Node.js project. Continue with manual port detection?`,
        actions: [
          { label: 'Cancel',            value: 'cancel',   style: 'btn-ghost' },
          { label: 'Continue Anyway',   value: 'continue', style: 'btn-primary' },
        ],
      })
      if (proceed !== 'continue') return
    }

    showLoadingModal('Scanning ports…', 'Reading active listeners')
    let snapshot = []
    try {
      snapshot = await Promise.race([
        GetListeningPorts(),
        new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), 8000)),
      ])
    } catch (_) { /* timeout/empty → detection loop still works */ }
    closeModal()

    detectedPorts = await detectionModeLoop(snapshot)
    if (detectedPorts === null) return   // user cancelled
  }

  // ── Step 3: Server Proposal for first detected port ───────────────────────
  const firstPort = detectedPorts[0]
  const s3 = await showModal({
    title: 'Server Proposal',
    message: 'Review the auto-detected configuration and confirm.',
    formContent: `
      <div class="space-y-2 mt-2">
        <input id="sp-name" type="text" value="${esc(suggestedScript)}" placeholder="Server name"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
        <input id="sp-cmd" type="text" value="${esc(suggestedCmd)}" placeholder="Run command (e.g. npm run dev)"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100">
        <input id="sp-port" type="number" value="${firstPort}" min="1024" max="65535"
          class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100">
        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
          <input type="checkbox" id="sp-auto" class="accent-indigo-500">
          Start on VPM launch
        </label>
      </div>
    `,
    actions: [
      { label: 'Cancel',                value: 'cancel', style: 'btn-ghost' },
      { label: 'Skip server for now',   value: 'skip',   style: 'btn-ghost' },
      { label: 'Add Project & Server',  value: 'add',    style: 'btn-primary' },
    ],
  })
  if (s3 === 'cancel') return

  try {
    const proj = await AddProject(path, displayName, null)

    if (s3 === 'add') {
      const srvName  = document.getElementById('sp-name')?.value?.trim() || suggestedScript || 'dev'
      const srvCmd   = document.getElementById('sp-cmd')?.value?.trim()
      const srvPort  = parseInt(document.getElementById('sp-port')?.value, 10)
      const srvAuto  = document.getElementById('sp-auto')?.checked ?? false

      if (!srvCmd) { await showError('Missing command', 'Please enter a run command.'); return }
      if (!srvPort || srvPort < 1024 || srvPort > 65535) {
        await showError('Invalid port', 'Port must be between 1024 and 65535.')
        return
      }

      await AddServer(proj.id, {
        id: '', name: srvName, command: srvCmd, port: srvPort, autostart: srvAuto,
      })

      // ── Step 4: offer additional ports ──────────────────────────────────────
      // 4a: leftover ports from detection mode (rare but possible)
      const remaining = detectedPorts.slice(1).filter(p => p !== srvPort)

      // 4b: any currently-active ports not yet in this project (3 s timeout)
      let liveExtra = []
      try {
        const livePorts = await Promise.race([
          GetListeningPorts(),
          new Promise(r => setTimeout(() => r([]), 3000)),
        ])
        const projNow  = (await ListProjects()).find(p => p.id === proj.id)
        const regPorts = new Set((projNow?.servers ?? []).map(s => s.port))
        const skip     = new Set([srvPort, ...remaining])
        liveExtra = livePorts.filter(p => !regPorts.has(p) && !skip.has(p) && p >= 1024 && p <= 49151)
      } catch (_) {}

      for (const port of [...remaining, ...liveExtra]) {
        const more = await showModal({
          title: 'Additional Port Detected',
          message: `Port :${port} is also active. Register it as a server for this project?`,
          formContent: `
            <div class="space-y-2 mt-2">
              <input id="mp-name" type="text" value="srv-${port}" placeholder="Server name"
                class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-gray-100">
              <input id="mp-cmd" type="text" value="${esc(srvCmd)}" placeholder="Command"
                class="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 mono text-sm text-gray-100">
            </div>
          `,
          actions: [
            { label: 'Skip',          value: 'skip', style: 'btn-ghost' },
            { label: 'Add Server →',  value: 'add',  style: 'btn-primary' },
          ],
        })
        if (more === 'add') {
          const mpName = document.getElementById('mp-name')?.value?.trim() || `srv-${port}`
          const mpCmd  = document.getElementById('mp-cmd')?.value?.trim() || srvCmd
          if (mpCmd) {
            await AddServer(proj.id, { id: '', name: mpName, command: mpCmd, port, autostart: false })
          }
        }
      }
    }

    const projects = await ListProjects()
    setProjects(projects)
    setProjectFilter(proj.id)   // navigate to new project
  } catch (e) {
    await showError('Failed to add', e?.message)
  }
}

// runMultiServerImport: register a project + multiple detected servers at once
// (from a PM2 ecosystem config). All selected servers share the autostart /
// autorestart flags chosen in the modal.
async function runMultiServerImport(path, displayName, servers) {
  const srcLabel = {
    workspace: 'workspace', pm2: 'PM2', compose: 'compose', package: 'package',
    python: 'python', go: 'go', rust: 'rust', procfile: 'Procfile', task: 'task',
    dotnet: '.NET', ruby: 'Ruby', php: 'PHP', java: 'Spring', elixir: 'Phoenix', deno: 'Deno',
  }
  const rows = servers.map((s, i) => `
    <div class="py-1.5 border-b border-gray-800 last:border-0">
      <div class="flex items-center gap-2">
        <input type="checkbox" name="imp-srv" data-idx="${i}" checked class="accent-indigo-500 shrink-0">
        <input type="text" data-name-idx="${i}" value="${esc(s.name)}"
          class="flex-1 min-w-0 bg-gray-800 border border-gray-600 rounded px-2 py-1 text-xs text-gray-100">
        <input type="number" data-port-idx="${i}" value="${s.port > 0 ? s.port : ''}"
          placeholder="port" min="1024" max="65535"
          class="w-20 shrink-0 bg-gray-800 border border-gray-600 rounded px-2 py-1 mono text-xs text-gray-100 placeholder-gray-600">
        <span class="text-[10px] text-gray-600 shrink-0 w-14 text-right">${esc(srcLabel[s.source] ?? s.source)}</span>
      </div>
      <input type="text" data-cmd-idx="${i}" value="${esc(s.command)}"
        class="w-full mt-1 bg-gray-800 border border-gray-600 rounded px-2 py-1 mono text-xs text-gray-300">
    </div>
  `).join('')

  const result = await showModal({
    title: `Import ${servers.length} services`,
    message: 'Select services to register. Fill in any missing ports (apps without a detected port).',
    formContent: `
      <div class="mt-2 max-h-64 overflow-y-auto">${rows}</div>
      <p class="text-xs text-gray-600 mt-1">Sources: ${[...new Set(servers.map(s => srcLabel[s.source] ?? s.source))].join(', ')}</p>
      <div class="mt-3 space-y-1.5 border-t border-gray-800 pt-2">
        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
          <input type="checkbox" id="imp-autostart" class="accent-indigo-500"> Start on VPM launch
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
          <input type="checkbox" id="imp-autorestart" checked class="accent-indigo-500"> Auto-restart on crash
        </label>
      </div>
    `,
    actions: [
      { label: 'Cancel', value: 'cancel', style: 'btn-ghost' },
      { label: 'Import', value: 'import', style: 'btn-primary' },
    ],
  })
  if (result !== 'import') return

  const autostart   = document.getElementById('imp-autostart')?.checked ?? false
  const autorestart = document.getElementById('imp-autorestart')?.checked ?? false

  const toAdd = []
  const skipped = []
  for (const cb of document.querySelectorAll('input[name="imp-srv"]:checked')) {
    const idx     = cb.dataset.idx
    const s       = servers[parseInt(idx, 10)]
    const name    = document.querySelector(`[data-name-idx="${idx}"]`)?.value?.trim()
    const command = document.querySelector(`[data-cmd-idx="${idx}"]`)?.value?.trim()
    const port    = parseInt(document.querySelector(`[data-port-idx="${idx}"]`)?.value, 10)
    const label   = name || s?.name || `#${idx}`
    if (!name || !command || !port || port < 1024 || port > 65535) { skipped.push(label); continue }
    toAdd.push({ name, command, port })
  }
  if (toAdd.length === 0) {
    await showError('Nothing to import', 'Each selected service needs a name, command, and a valid port (1024–65535).')
    return
  }

  try {
    const proj = await AddProject(path, displayName, null)
    const failed = []
    for (const s of toAdd) {
      try {
        await AddServer(proj.id, { id: '', name: s.name, command: s.command, port: s.port, autostart, autorestart })
      } catch (e) { failed.push(`${s.name} (${e?.message ?? 'error'})`) }
    }
    const projects = await ListProjects()
    setProjects(projects)
    setProjectFilter(proj.id)

    const notes = []
    if (skipped.length) notes.push(`Skipped (need name/command/port): ${skipped.join(', ')}`)
    if (failed.length) notes.push(`Failed: ${failed.join(', ')}`)
    if (notes.length) await showModal({ title: 'Imported with notes', message: notes.join('\n') })
  } catch (e) {
    await showError('Import failed', e?.message)
  }
}

// detectionModeLoop: take snapshot, ask user to start server, re-scan, diff, pick candidates.
// Returns int[] of selected ports, or null if cancelled.
async function detectionModeLoop(snapshot) {
  const snapshotSet = new Set(snapshot)

  while (true) {
    const scan = await showModal({
      title: 'Detection Mode',
      message: 'Port not detected automatically. Start your dev server in a terminal, then click Scan.',
      formContent: `
        <p class="text-xs text-gray-500 mt-2">
          Baseline: ${snapshot.length} port${snapshot.length !== 1 ? 's' : ''} active.
          New ports that appear after scanning will be shown as candidates.
        </p>
      `,
      actions: [
        { label: 'Cancel',             value: 'cancel', style: 'btn-ghost' },
        { label: 'Scan for New Ports', value: 'scan',   style: 'btn-primary' },
      ],
    })
    if (scan !== 'scan') return null

    let current = []
    try { current = await GetListeningPorts() } catch (_) {}

    const candidates = current.filter(p => !snapshotSet.has(p))

    if (candidates.length === 0) {
      const retry = await showModal({
        title: 'No New Ports Found',
        message: 'No new listening ports detected. Start your dev server in a terminal, then scan again.',
        actions: [
          { label: 'Cancel',     value: 'cancel', style: 'btn-ghost' },
          { label: 'Try Again',  value: 'retry',  style: 'btn-primary' },
        ],
      })
      if (retry !== 'retry') return null
      continue
    }

    // Single candidate: return immediately as array
    if (candidates.length === 1) return candidates

    // Multiple candidates: checkboxes (all pre-checked — register all by default)
    const pick = await showModal({
      title: 'Select Ports',
      message: 'Multiple new ports found. Select all servers to register:',
      formContent: `
        <div class="space-y-1.5 mt-2">
          ${candidates.map(p => `
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" name="port-check" value="${p}"
                checked class="accent-indigo-500">
              <span class="mono text-sm text-gray-200">:${p}</span>
            </label>
          `).join('')}
        </div>
      `,
      actions: [
        { label: 'Cancel',       value: 'cancel', style: 'btn-ghost' },
        { label: 'Register →',   value: 'next',   style: 'btn-primary' },
      ],
    })
    if (pick !== 'next') return null

    const checked = [...document.querySelectorAll('input[name="port-check"]:checked')]
    const selected = checked.map(cb => parseInt(cb.value, 10)).filter(Boolean)
    return selected.length > 0 ? selected : null
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

