// components/alert-panel.js — inline collision alert strip
// Listens to vpm:collision, slides in above view-content.
// When the backend supplies a suggested free port, offers a one-click apply
// that dispatches vpm:apply-suggested-port (handled by views/server-list.js).

export function init(el) {
  el.className = 'hidden shrink-0 bg-red-950 border-b border-red-800 px-4 py-2'

  window.addEventListener('vpm:collision', e => show(el, e.detail))
}

function show(el, payload) {
  const desc       = payload.blockingDescription ? ` — ${payload.blockingDescription}` : ''
  const serverID   = payload.serverID
  const suggested  = payload.suggestedFreePort
  const canApply   = serverID && suggested && suggested > 0

  el.innerHTML = `
    <div class="flex items-center gap-3 text-xs">
      <span class="text-red-400 font-bold shrink-0">⚠ Port Collision</span>
      <span class="text-red-300 flex-1">
        PID ${payload.blockingPID}${desc} blocking port. Origin: ${esc(payload.origin ?? '')}.
      </span>
      ${canApply ? `
        <button id="ap-apply"
          class="shrink-0 px-2 py-0.5 rounded bg-red-700 hover:bg-red-600 text-white font-medium transition-colors">
          Use free port :${suggested}
        </button>` : ''}
      <button id="ap-dismiss" class="btn-ghost text-xs py-0.5 shrink-0">Dismiss</button>
    </div>
  `
  el.classList.remove('hidden')

  el.querySelector('#ap-dismiss')?.addEventListener('click', () => dismiss(el))

  el.querySelector('#ap-apply')?.addEventListener('click', () => {
    window.dispatchEvent(new CustomEvent('vpm:apply-suggested-port', {
      detail: { serverID, port: suggested },
    }))
    dismiss(el)
  })
}

function dismiss(el) {
  el.classList.add('hidden')
  el.innerHTML = ''
}

function esc(str) {
  return String(str ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
