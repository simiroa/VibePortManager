// modal.js — alert() replacement using Tailwind overlay
// Never call native alert(); use showModal() instead.

let _resolve

// showLoadingModal shows a spinner modal with no buttons.
// Call closeModal() when the async work finishes.
export function showLoadingModal(title, message) {
  const overlay = document.getElementById('modal-overlay')
  document.getElementById('modal-title').textContent = title
  const msgEl = document.getElementById('modal-message')
  msgEl.textContent = message ?? ''
  const slot = document.getElementById('modal-form-slot')
  if (slot) slot.innerHTML = `
    <div class="flex flex-col items-center gap-3 py-4">
      <svg class="w-8 h-8 animate-spin text-indigo-400" viewBox="0 0 24 24" fill="none">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
      </svg>
      <div id="loading-sub" class="text-xs text-gray-600"></div>
    </div>
  `
  document.getElementById('modal-actions').innerHTML = ''
  overlay.classList.remove('hidden')
}

// updateLoadingStatus updates the sub-label inside a showLoadingModal.
export function updateLoadingStatus(text) {
  const el = document.getElementById('loading-sub')
  if (el) el.textContent = text
}

// closeModal forcibly hides the overlay and resolves any pending showModal promise.
export function closeModal() {
  document.getElementById('modal-overlay').classList.add('hidden')
  if (_resolve) { _resolve(null); _resolve = null }
}

export function showModal({ title, message, formContent, actions = [{ label: 'OK', value: true, style: 'btn-ghost' }] }) {
  return new Promise(resolve => {
    _resolve = resolve
    const overlay = document.getElementById('modal-overlay')
    document.getElementById('modal-title').textContent = title
    document.getElementById('modal-message').textContent = message ?? ''

    const slot = document.getElementById('modal-form-slot')
    if (slot) slot.innerHTML = formContent ?? ''

    const btns = document.getElementById('modal-actions')
    btns.innerHTML = ''
    for (const a of actions) {
      const btn = document.createElement('button')
      btn.className = a.style || 'btn-ghost'
      btn.textContent = a.label
      btn.addEventListener('click', () => {
        overlay.classList.add('hidden')
        _resolve(a.value)
      })
      btns.appendChild(btn)
    }

    overlay.classList.remove('hidden')
  })
}

export function showError(title, message) {
  return showModal({
    title,
    message,
    actions: [{ label: 'OK', value: true, style: 'btn-ghost' }]
  })
}

export function showConfirm(title, message) {
  return showModal({
    title,
    message,
    actions: [
      { label: 'Cancel', value: false, style: 'btn-ghost' },
      { label: 'Confirm', value: true, style: 'btn-red' },
    ]
  })
}
