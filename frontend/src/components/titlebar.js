// components/titlebar.js — custom dark titlebar for the frameless window.
// Provides drag region + window controls (mini / minimise / maximise / close).

import { WindowMinimise, WindowToggleMaximise, QuitApp } from '../wails.js'
import { enterMiniMode } from './mini-bar.js'

export function init(el) {
  el.className = 'shrink-0 flex items-stretch h-9 bg-gray-900 border-b border-gray-800 select-none'
  el.style.setProperty('--wails-draggable', 'drag') // whole bar drags the window

  el.innerHTML = `
    <div class="flex items-center gap-2 px-3 flex-1 min-w-0">
      <span class="text-indigo-400 text-sm leading-none">⚡</span>
      <span class="text-xs font-semibold text-gray-300 truncate">Vibe Port Manager</span>
    </div>
    <div class="flex items-stretch" style="--wails-draggable:no-drag">
      <button data-win="mini"  title="Mini mode"  class="tb-btn">▬</button>
      <button data-win="min"   title="Minimise"   class="tb-btn">─</button>
      <button data-win="max"   title="Maximise"   class="tb-btn">▢</button>
      <button data-win="close" title="Close"      class="tb-btn tb-close">✕</button>
    </div>
  `

  el.querySelector('[data-win="mini"]').addEventListener('click', () => enterMiniMode())
  el.querySelector('[data-win="min"]').addEventListener('click', () => WindowMinimise())
  el.querySelector('[data-win="max"]').addEventListener('click', () => WindowToggleMaximise())
  el.querySelector('[data-win="close"]').addEventListener('click', () => QuitApp())
}
