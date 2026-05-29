// components/log-strip.js — bottom log strip: render + live append + scroll
// Concept: all log-strip rendering and live-update logic in one place.

import { findProjectByServer } from '../store.js'
import { getSelectedServerID, getLogLines } from '../state.js'

// ── Public render ─────────────────────────────────────────────────────────────

export function renderLogStrip(logExpanded, newCount = 0) {
  const serverID = getSelectedServerID()
  const info     = serverID ? findProjectByServer(serverID) : null
  const label    = info ? `${info.project.name} / ${info.server.name}` : 'No server selected'
  const bodyClass = logExpanded ? 'h-44 border-t border-gray-800' : 'hidden'

  return `
    <div id="log-strip" class="shrink-0 bg-gray-950">
      <div id="log-strip-header"
        class="flex items-center gap-2 px-4 py-1.5 border-t border-gray-800 cursor-pointer
               hover:bg-gray-900 transition-colors select-none">
        <span class="text-xs text-gray-500 flex-1 truncate">
          <span class="text-indigo-400 mr-1">▪</span>${esc(label)}
        </span>
        ${!logExpanded && newCount > 0 ? `
          <span id="log-new-count"
            class="text-xs text-indigo-300 bg-indigo-900/50 rounded px-1.5 py-0.5 shrink-0">
            ●${newCount} new
          </span>
        ` : '<span id="log-new-count" class="hidden"></span>'}
        ${serverID ? `
          <button data-action="clear-log" data-server-id="${esc(serverID)}"
            class="text-xs text-gray-600 hover:text-gray-400 transition-colors px-1">
            Clear
          </button>
          <button data-action="export-log" data-server-id="${esc(serverID)}"
            class="text-xs text-gray-600 hover:text-gray-400 transition-colors px-1">
            Export…
          </button>
        ` : ''}
        <span class="text-gray-600 text-xs">${logExpanded ? '▼' : '▲'}</span>
      </div>

      <div id="log-body" class="${bodyClass} overflow-y-auto bg-gray-950 mono text-xs text-green-400 px-4 py-2">
        ${serverID
          ? getLogLines(serverID).map(l => `<div>${escLog(l)}</div>`).join('') || '<span class="text-gray-700">— no output yet —</span>'
          : '<span class="text-gray-700">Click a server card to view its logs.</span>'
        }
      </div>
    </div>
  `
}

// ── Public live-update helpers ────────────────────────────────────────────────

export function appendLogLinesDOM(containerEl, lines) {
  const body = containerEl?.querySelector('#log-body')
  if (!body) return
  lines.forEach(l => {
    const div = document.createElement('div')
    div.textContent = l
    body.appendChild(div)
  })
  scrollLogToBottom(containerEl)
}

export function scrollLogToBottom(containerEl) {
  const body = containerEl?.querySelector('#log-body')
  if (body) body.scrollTop = body.scrollHeight
}

// ── Private helpers ───────────────────────────────────────────────────────────

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function escLog(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
