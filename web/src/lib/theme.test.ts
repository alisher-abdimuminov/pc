import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { _resetThemeForTests, applyTheme, initialTheme, setTheme, useTheme } from './theme'

describe('initialTheme', () => {
  it('prefers the saved choice over the system', () => {
    expect(initialTheme('light', true)).toBe('light')
    expect(initialTheme('dark', false)).toBe('dark')
  })
  it('falls back to the system preference, ignoring garbage', () => {
    expect(initialTheme(null, true)).toBe('dark')
    expect(initialTheme(null, false)).toBe('light')
    expect(initialTheme('purple', true)).toBe('dark')
  })
})

describe('applyTheme', () => {
  it('toggles the dark class and the colour scheme', () => {
    const root = document.createElement('html')
    applyTheme('dark', root)
    expect(root.classList.contains('dark')).toBe(true)
    expect(root.style.colorScheme).toBe('dark')
    applyTheme('light', root)
    expect(root.classList.contains('dark')).toBe(false)
    expect(root.style.colorScheme).toBe('light')
  })
})

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    _resetThemeForTests()
  })
  afterEach(() => vi.unstubAllGlobals())

  it('starts from the system preference and applies it', () => {
    vi.stubGlobal('matchMedia', () => ({ matches: true, addEventListener: vi.fn() }))
    expect(useTheme().theme.value).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('restores a saved choice', () => {
    localStorage.setItem('theme', 'light')
    vi.stubGlobal('matchMedia', () => ({ matches: true, addEventListener: vi.fn() }))
    expect(useTheme().theme.value).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('saves and applies a new choice', () => {
    vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener: vi.fn() }))
    const { theme } = useTheme()
    setTheme('dark')
    expect(theme.value).toBe('dark')
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('follows the system until the user chooses, and not after', () => {
    let notify: (e: { matches: boolean }) => void = () => undefined
    vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener: (_: string, cb: typeof notify) => (notify = cb) }))
    const { theme } = useTheme()
    notify({ matches: true })
    expect(theme.value).toBe('dark')
    expect(localStorage.getItem('theme')).toBeNull() // following the system is not a choice
    setTheme('light')
    notify({ matches: true })
    expect(theme.value).toBe('light')
  })

  it('still works when storage is blocked', () => {
    vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener: vi.fn() }))
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    const { theme } = useTheme()
    expect(() => setTheme('dark')).not.toThrow()
    expect(theme.value).toBe('dark')
    vi.restoreAllMocks()
  })
})
