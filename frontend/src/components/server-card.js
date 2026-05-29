// components/server-card.js — server card and add-server card rendering
// Concept: pure render functions for a single server card.
// State reads (serverState, startTime, selection) are imported directly.

import { getServerState, getServerStartTime, getServerError } from '../store.js'
import { getSelectedServerID } from '../state.js'

// ── Public render functions ───────────────────────────────────────────────────

export function renderServerListItem(srv, projectID) {
  const state     = getServerState(srv.id)
  const running   = state === 'RUNNING'
  const collision = state === 'PORT_COLLISION'
  const errMsg    = (state === 'ERROR' || collision) ? getServerError(srv.id) : null
  const uptime    = running ? _formatUptime(getServerStartTime(srv.id)) : null
  const cmdHint   = srv.command ? _shortCmd(srv.command) : null

  // Status dot color
  const dotColor = state === 'RUNNING' ? 'bg-green-500'
    : state === 'ERROR' || state === 'PORT_COLLISION' ? 'bg-red-500'
    : state === 'STARTING' ? 'bg-yellow-500'
    : state === 'STOPPING' ? 'bg-orange-500'
    : 'bg-gray-500'

  const dotClass = `inline-block w-2 h-2 rounded-full shrink-0 ${dotColor}`

  return `
    <div data-action="select-server" data-server-id="${esc(srv.id)}"
      class="group cursor-pointer flex items-center gap-2 px-3 py-2 rounded-lg
             border border-gray-800 hover:border-gray-700 hover:bg-gray-900/50
             transition-all">

      <!-- Status dot -->
      <span class="${dotClass}" title="${state}"></span>

      <!-- Port (primary identifier) -->
      <span class="mono font-bold text-gray-100 w-16 shrink-0">:${srv.port}</span>

      <!-- Name -->
      <span class="text-xs font-medium text-gray-300 w-20 truncate shrink-0">${esc(srv.name)}</span>

      <!-- Command preview -->
      <span class="mono text-xs text-gray-600 flex-1 truncate min-w-0">${cmdHint ? esc(cmdHint) : ''}</span>

      <!-- Uptime / status -->
      <span class="mono text-xs text-gray-600 w-12 text-right shrink-0">
        ${uptime ? esc(uptime) : '—'}
      </span>

      <!-- Actions (hidden until hover) -->
      <div class="flex gap-1 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
        <button data-action="start" data-server-id="${esc(srv.id)}"
          ${running ? 'disabled' : ''} title="Start"
          class="w-5 h-5 flex items-center justify-center rounded text-gray-400 hover:text-green-400
                 hover:bg-green-900/30 transition-colors text-sm ${running ? 'opacity-30' : ''}" disabled="${running}">▶</button>
        <button data-action="stop" data-server-id="${esc(srv.id)}"
          ${!running ? 'disabled' : ''} title="Stop"
          class="w-5 h-5 flex items-center justify-center rounded text-gray-400 hover:text-red-400
                 hover:bg-red-900/30 transition-colors text-sm ${!running ? 'opacity-30' : ''}" disabled="${!running}">■</button>
        <button data-action="restart" data-server-id="${esc(srv.id)}"
          ${!running ? 'disabled' : ''} title="Restart"
          class="w-5 h-5 flex items-center justify-center rounded text-gray-400 hover:text-gray-300
                 hover:bg-gray-700 transition-colors text-xs ${!running ? 'opacity-30' : ''}" disabled="${!running}">↺</button>
        ${running ? `
          <button data-action="open-browser" data-server-id="${esc(srv.id)}" data-port="${srv.port}"
            title="Open in browser"
            class="w-5 h-5 flex items-center justify-center rounded text-gray-400 hover:text-indigo-400
                   hover:bg-indigo-900/30 transition-colors text-sm">↗</button>
        ` : ''}
        <button data-action="open-card-menu"
          data-server-id="${esc(srv.id)}" data-project-id="${esc(projectID)}"
          class="w-5 h-5 flex items-center justify-center rounded text-gray-400 hover:text-gray-200
                 hover:bg-gray-700 transition-colors text-sm leading-none">⋮</button>
      </div>

      ${errMsg ? `<div class="absolute top-full left-0 mt-1 text-xs text-red-400 bg-gray-800 rounded px-2 py-1
                    hidden group-hover:block whitespace-nowrap z-10">${esc(errMsg)}</div>` : ''}
    </div>
  `
}

export function renderServerCard(srv, projectID) {
  const state     = getServerState(srv.id)
  const selected  = getSelectedServerID() === srv.id
  const busy      = state === 'STARTING' || state === 'STOPPING'
  const running   = state === 'RUNNING'
  const collision = state === 'PORT_COLLISION'
  const errMsg    = (state === 'ERROR' || collision) ? getServerError(srv.id) : null

  const badgeClass = _stateColors[state] ?? _stateColors.STOPPED
  const cardBorder = selected
    ? 'border-indigo-500/60 bg-gray-900'
    : 'border-gray-700/50 bg-gray-900 hover:border-gray-600'

  const uptime  = running ? _formatUptime(getServerStartTime(srv.id)) : null
  const cmdHint = srv.command ? _shortCmd(srv.command) : null

  return `
    <div data-action="select-server" data-server-id="${esc(srv.id)}"
      class="cursor-pointer rounded-xl border p-3 transition-all ${cardBorder}">

      <!-- Top row: state badge + uptime + ⋮ menu -->
      <div class="flex items-center gap-1.5 mb-2">
        <span class="shrink-0 text-xs px-1.5 py-0.5 rounded border ${badgeClass} flex items-center gap-1">
          ${busy ? _spinnerSvg() : ''}<span>${state}</span>
        </span>
        ${uptime ? `<span class="mono text-xs text-gray-600">${uptime}</span>` : ''}
        <div class="ml-auto relative">
          <button data-action="open-card-menu"
            data-server-id="${esc(srv.id)}" data-project-id="${esc(projectID)}"
            class="w-6 h-5 flex items-center justify-center rounded text-gray-600
                   hover:text-gray-300 hover:bg-gray-700 transition-colors text-base leading-none select-none"
            title="More actions">⋮</button>
        </div>
      </div>

      <!-- Port — primary identifier -->
      <p class="mono text-lg font-bold text-gray-100 leading-none mb-1">:${srv.port}</p>

      <!-- Name + command preview -->
      <p class="text-xs text-gray-500 truncate ${errMsg ? 'mb-1' : 'mb-3'}">
        <span class="text-gray-400 font-medium">${esc(srv.name)}</span>
        ${cmdHint ? `<span class="mono text-gray-600 ml-1.5">${esc(cmdHint)}</span>` : ''}
      </p>

      <!-- Error message (item 3) -->
      ${errMsg ? `<p class="text-xs text-red-400 truncate mb-3" title="${esc(errMsg)}">${esc(errMsg)}</p>` : ''}

      <!-- Lifecycle buttons -->
      <div class="flex gap-1.5 flex-wrap">
        <button data-action="start" data-server-id="${esc(srv.id)}"
          ${running || busy ? 'disabled' : ''} title="Start"
          class="btn-xs ${!running && !busy ? 'btn-green' : 'btn-disabled'}">▶ Start</button>
        <button data-action="stop" data-server-id="${esc(srv.id)}"
          ${!running || busy ? 'disabled' : ''} title="Stop"
          class="btn-xs ${running && !busy ? 'btn-red' : 'btn-disabled'}">■ Stop</button>
        <button data-action="restart" data-server-id="${esc(srv.id)}"
          ${!running || busy ? 'disabled' : ''} title="Restart"
          class="btn-xs ${running && !busy ? 'btn-gray' : 'btn-disabled'}">↺</button>
        ${running ? `
          <button data-action="open-browser" data-server-id="${esc(srv.id)}" data-port="${srv.port}"
            title="Open http://localhost:${srv.port} in browser"
            class="btn-xs btn-gray">↗ Open</button>
        ` : ''}
        ${collision ? `
          <button data-action="suggest-port" data-server-id="${esc(srv.id)}" data-port="${srv.port}"
            title="Change to a free port"
            class="btn-xs btn-yellow">Change Port</button>
        ` : ''}
      </div>
    </div>
  `
}

export function renderAddServerCard(projectID) {
  return `
    <button data-action="add-server" data-project-id="${esc(projectID)}"
      class="cursor-pointer px-3 py-2 rounded-lg border border-dashed border-gray-700
             text-gray-600 text-xs font-medium hover:border-indigo-500 hover:text-indigo-400
             transition-colors w-full text-left">
      + Add server
    </button>
  `
}

// ── Private helpers ───────────────────────────────────────────────────────────

const _stateColors = {
  RUNNING:        'text-green-400 bg-green-900/30 border-green-800/50',
  STARTING:       'text-yellow-400 bg-yellow-900/30 border-yellow-800/50',
  STOPPING:       'text-orange-400 bg-orange-900/30 border-orange-800/50',
  ERROR:          'text-red-400 bg-red-900/30 border-red-800/50',
  PORT_COLLISION: 'text-red-300 bg-red-900/40 border-red-700/60',
  STOPPED:        'text-gray-500 bg-gray-800/40 border-gray-700/40',
}

function _formatUptime(startTime) {
  if (!startTime) return null
  const m = Math.floor((Date.now() - startTime) / 60000)
  const h = Math.floor(m / 60)
  if (h > 0) return `↑ ${h}h ${m % 60}m`
  if (m > 0) return `↑ ${m}m`
  return '↑ <1m'
}

function _shortCmd(cmd) {
  const parts = cmd.trim().split(/\s+/)
  // Show up to 3 tokens; trim long commands to keep card compact
  const short = parts.slice(0, 3).join(' ')
  return short.length > 28 ? short.slice(0, 26) + '…' : short
}

function _spinnerSvg() {
  return `<svg class="inline w-3 h-3 animate-spin" viewBox="0 0 24 24" fill="none">
    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
  </svg>`
}

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}
