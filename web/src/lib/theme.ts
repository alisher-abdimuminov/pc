import { ref } from 'vue'

export type Theme = 'light' | 'dark'

const KEY = 'theme'

/** The saved choice wins; without one the system preference decides. */
export function initialTheme(stored: string | null, prefersDark: boolean): Theme {
  if (stored === 'light' || stored === 'dark') return stored
  return prefersDark ? 'dark' : 'light'
}

export function applyTheme(theme: Theme, root: HTMLElement = document.documentElement) {
  root.classList.toggle('dark', theme === 'dark')
  root.style.colorScheme = theme
}

function readStored(): string | null {
  try {
    return localStorage.getItem(KEY)
  } catch {
    return null // storage can be blocked (private mode)
  }
}

const theme = ref<Theme>('light')
let started = false

function systemPrefersDark() {
  return typeof matchMedia === 'function' && matchMedia('(prefers-color-scheme: dark)').matches
}

/** Shared theme state. The first call reads the saved choice and starts following the system until the user picks. */
export function useTheme() {
  if (!started) {
    started = true
    theme.value = initialTheme(readStored(), systemPrefersDark())
    applyTheme(theme.value)
    if (typeof matchMedia === 'function') {
      matchMedia('(prefers-color-scheme: dark)').addEventListener?.('change', (e) => {
        if (readStored() === null) setTheme(e.matches ? 'dark' : 'light', false)
      })
    }
  }
  return { theme, setTheme }
}

export function setTheme(next: Theme, persist = true) {
  theme.value = next
  applyTheme(next)
  if (persist) {
    try {
      localStorage.setItem(KEY, next)
    } catch {
      /* the choice still applies for this visit */
    }
  }
}

/** Test helper: forget module state. */
export function _resetThemeForTests() {
  started = false
}
