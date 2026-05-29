// frontend/src/theme.js — Light/Dark mode theme management
// Provides theme initialization, toggle, and OS preference detection

import { WindowSetBackgroundColour } from './wails.js'

export function initTheme() {
  // Listen for OS theme preference changes
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', e => {
    // Only auto-switch if user hasn't manually overridden
    if (!localStorage.getItem('vpm-theme')) {
      applyTheme(e.matches ? 'dark' : 'light')
    }
  })

  // Dispatch theme-changed event for titlebar to update
  window.dispatchEvent(new CustomEvent('vpm:theme-changed', { detail: isDark() }))
}

export function toggleTheme() {
  const next = isDark() ? 'light' : 'dark'
  localStorage.setItem('vpm-theme', next)
  applyTheme(next)
}

function applyTheme(theme) {
  const isDarkTheme = theme === 'dark'
  document.documentElement.classList.toggle('dark', isDarkTheme)

  // Sync window background color to prevent flash on theme switch
  if (isDarkTheme) {
    WindowSetBackgroundColour(10, 15, 26, 255)     // gray-950: #0a0f1a
  } else {
    WindowSetBackgroundColour(248, 250, 252, 255)  // slate-50: #f8fafc
  }

  window.dispatchEvent(new CustomEvent('vpm:theme-changed', { detail: isDarkTheme }))
}

export function isDark() {
  return document.documentElement.classList.contains('dark')
}
