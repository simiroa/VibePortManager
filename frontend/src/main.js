// main.js — app boot: shell layout, event wiring, sidebar + single server-list view
import './main.css'
import { setProjects, applyStateChange, getProjects, getServerState } from './store.js'
import { appendLogLines } from './state.js'
import { onEvent, GetSystemStats, ListProjects, GetAllServerStates } from './wails.js'

// ── Shell ─────────────────────────────────────────────────────────────────────

function renderShell() {
  document.body.innerHTML = `
    <!-- Modal overlay -->
    <div id="modal-overlay" class="hidden fixed inset-0 z-50 flex items-center justify-center bg-black/70">
      <div class="bg-gray-900 border border-gray-700 rounded-xl shadow-2xl p-6 w-full max-w-md mx-4">
        <h3 id="modal-title" class="text-base font-bold text-gray-100 mb-2"></h3>
        <p id="modal-message" class="text-sm text-gray-400 mb-1"></p>
        <div id="modal-form-slot"></div>
        <div id="modal-actions" class="flex gap-2 justify-end mt-4"></div>
      </div>
    </div>

    <!-- Mini mode: compact floating status bar (hidden unless active) -->
    <div id="mini-bar" class="hidden"></div>

    <!-- Full app: titlebar + (sidebar + content column) -->
    <div id="full-app" class="flex flex-col h-screen bg-gray-950 overflow-hidden">

      <!-- Custom dark titlebar (frameless window) -->
      <div id="titlebar"></div>

      <div class="flex flex-1 min-h-0 overflow-hidden">

        <!-- Sidebar -->
        <aside class="w-52 shrink-0 flex flex-col border-r border-gray-800 bg-gray-900 overflow-hidden">
          <div id="sidebar-shell" class="flex-1 flex flex-col min-h-0"></div>
          <div id="status-bar" class="shrink-0 border-t border-gray-800 px-3 py-2 text-xs text-gray-500 space-y-0.5">
            <div id="stat-running">— servers running</div>
            <div id="stat-ram">— MB RAM</div>
            <div id="stat-errors"></div>
          </div>
        </aside>

        <!-- Right column: collision alert + server list view -->
        <div class="flex-1 flex flex-col overflow-hidden">
          <div id="alert-panel" class="shrink-0 hidden"></div>
          <main id="view-content" class="flex-1 overflow-hidden flex flex-col min-h-0"></main>
        </div>

      </div>
    </div>
  `
}

// ── Wails event wiring ────────────────────────────────────────────────────────

function wireEvents() {
  onEvent('server.state.changed', payload => {
    applyStateChange(payload)
  })

  onEvent('server.log.batch', payload => {
    appendLogLines(payload.serverID, payload.lines ?? [])
    window.dispatchEvent(new CustomEvent('vpm:log-batch', { detail: payload }))
  })

  onEvent('collision.detected', payload => {
    window.dispatchEvent(new CustomEvent('vpm:collision', { detail: payload }))
  })

  onEvent('tray.firstHide', () => {
    import('./modal.js').then(m => m.showModal({
      title: 'Minimised to Tray',
      message: 'VPM is still running with your servers. Use the tray icon in the notification area — "Show VPM" to restore, or "Quit" to stop everything and exit.',
      actions: [{ label: 'Got it', value: true, style: 'btn-ghost' }],
    }))
  })
}

// ── Stats poller (5 s) ────────────────────────────────────────────────────────

async function startStatsPoller() {
  const tick = async () => {
    try {
      const stats = await GetSystemStats()
      const r   = document.getElementById('stat-running')
      const ram = document.getElementById('stat-ram')
      const err = document.getElementById('stat-errors')
      if (r)   r.textContent   = `${stats.runningCount ?? 0} running`
      if (ram) ram.textContent = `${Math.round(stats.ramMB ?? 0)} MB RAM`
      // Count ERROR + PORT_COLLISION client-side (item 6)
      let errCount = 0
      for (const p of getProjects()) {
        for (const s of p.servers ?? []) {
          const st = getServerState(s.id)
          if (st === 'ERROR' || st === 'PORT_COLLISION') errCount++
        }
      }
      if (err) {
        err.textContent = errCount > 0 ? `${errCount} error${errCount > 1 ? 's' : ''}` : ''
        err.className   = errCount > 0 ? 'text-red-400' : ''
      }
    } catch (_) {}
  }
  await tick()
  setInterval(tick, 5000)
}

// ── Boot ──────────────────────────────────────────────────────────────────────

async function boot() {
  renderShell()
  wireEvents()

  // Load initial projects + sync server states from backend
  try {
    const [projects, states] = await Promise.all([ListProjects(), GetAllServerStates()])
    setProjects(projects)
    // Apply each state so cards reflect already-running servers immediately.
    for (const [serverID, state] of Object.entries(states ?? {})) {
      applyStateChange({ serverID, state })
    }
  } catch (e) {
    console.error('boot sync failed', e)
  }

  // Mount custom titlebar + mini-mode bar (frameless window chrome)
  import('./components/mini-bar.js').then(m => m.init(document.getElementById('mini-bar'))).catch(() => {})
  import('./components/titlebar.js').then(m => m.init(document.getElementById('titlebar'))).catch(() => {})

  // Mount sidebar (All + project list + Add Project)
  const sidebarMod = await import('./components/sidebar.js')
  sidebarMod.init(document.getElementById('sidebar-shell'))

  // Mount collision alert panel
  import('./components/alert-panel.js').then(m => {
    m.init(document.getElementById('alert-panel'))
  }).catch(() => {})

  // Mount unified server-list view (single view, no routing)
  const slMod = await import('./views/server-list.js')
  await slMod.init(document.getElementById('view-content'))

  startStatsPoller()
}

// window.go and window.runtime are injected by the Wails WebView2 host.
// They are NOT available in a plain browser — only in the Wails app window.
if (window.runtime) {
  boot()
} else {
  // Not running inside Wails — show a placeholder so the page isn't blank.
  document.addEventListener('DOMContentLoaded', () => {
    document.body.innerHTML =
      '<div style="display:flex;align-items:center;justify-content:center;height:100vh;' +
      'background:#030712;color:#4b5563;font-family:monospace;font-size:13px;">' +
      'VPM — open in the Wails app window (wails dev), not a browser.</div>'
  })
}
