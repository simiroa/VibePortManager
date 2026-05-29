// components/mini-bar.js — compact floating status bar.
// Shows only active servers as colour-coded port chips. Drag to move; click a
// chip to expand back to the full UI on that server; ⤢ to expand.

import { getProjects, getServerState } from '../store.js'
import { setProjectFilter, setSelectedServer } from '../state.js'
import { WindowGetSize, WindowSetSize, WindowSetAlwaysOnTop } from '../wails.js'

const MINI_W = 480
const MINI_H = 44

let _active = false
let _saved = null // { w, h } before entering mini mode

export function isMini() { return _active }

export function init(el) {
  el.className = 'hidden'
  const onChange = () => { if (_active) render() }
  window.addEventListener('vpm:state-changed', onChange)
  window.addEventListener('vpm:projects-updated', onChange)
}

export async function enterMiniMode() {
  if (_active) return
  try { _saved = await WindowGetSize() } catch (_) { _saved = { w: 1100, h: 720 } }
  _active = true
  document.getElementById('full-app')?.classList.add('hidden')
  document.getElementById('mini-bar')?.classList.remove('hidden')
  render()
  try {
    await WindowSetAlwaysOnTop(true)
    await WindowSetSize(MINI_W, MINI_H)
  } catch (_) {}
}

export async function exitMiniMode() {
  if (!_active) return
  _active = false
  try {
    await WindowSetAlwaysOnTop(false)
    if (_saved) await WindowSetSize(_saved.w, _saved.h)
  } catch (_) {}
  document.getElementById('mini-bar')?.classList.add('hidden')
  document.getElementById('full-app')?.classList.remove('hidden')
}

const MINI_COLORS = {
  RUNNING:        'bg-green-600 text-white',
  STARTING:       'bg-yellow-600 text-gray-950',
  STOPPING:       'bg-orange-600 text-white',
  ERROR:          'bg-red-600 text-white',
  PORT_COLLISION: 'bg-red-700 text-white',
}

function render() {
  const bar = document.getElementById('mini-bar')
  if (!bar) return
  bar.className = 'flex items-center h-screen w-screen bg-gray-950 overflow-hidden select-none'
  bar.style.setProperty('--wails-draggable', 'drag')

  const chips = []
  for (const p of getProjects()) {
    for (const s of (p.servers ?? [])) {
      const st = getServerState(s.id)
      if (!MINI_COLORS[st]) continue // active states only (skip STOPPED)
      chips.push(chipHTML(s, st))
    }
  }

  bar.innerHTML = `
    <span class="shrink-0 pl-2.5 pr-1 text-indigo-400 text-sm" title="Vibe Port Manager">⚡</span>
    <div class="flex items-center gap-1.5 px-1 flex-1 overflow-x-auto">
      ${chips.length ? chips.join('') : '<span class="text-xs text-gray-600">no active servers</span>'}
    </div>
    <button data-mini="expand" title="Expand to full view"
      class="shrink-0 h-full px-3 text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
      style="--wails-draggable:no-drag">⤢</button>
  `

  bar.querySelector('[data-mini="expand"]').addEventListener('click', () => exitMiniMode())
  bar.querySelectorAll('[data-mini-srv]').forEach(c => {
    c.addEventListener('click', async () => {
      const sid = c.dataset.miniSrv
      const info = getProjects().flatMap(p => (p.servers ?? []).map(s => ({ p, s }))).find(x => x.s.id === sid)
      if (info) {
        setProjectFilter(info.p.id)
        setSelectedServer(sid)
      }
      await exitMiniMode()
    })
  })
}

function chipHTML(s, st) {
  const cls = MINI_COLORS[st] ?? 'bg-gray-700 text-gray-200'
  return `<button data-mini-srv="${esc(s.id)}" title="${esc(s.name)} — ${st}"
    style="--wails-draggable:no-drag"
    class="shrink-0 mono text-xs font-bold px-2 py-1 rounded ${cls}">:${esc(s.port)}</button>`
}

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}
