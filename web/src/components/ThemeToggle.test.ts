import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { _resetThemeForTests } from '@/lib/theme'
import ThemeToggle from './ThemeToggle.vue'

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
  _resetThemeForTests()
  vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener: vi.fn() }))
})

describe('theme toggle', () => {
  it('shows a sun and a moon, with the current theme selected', () => {
    const w = mount(ThemeToggle)
    const [sun, moon] = w.findAll('[role="radio"]')
    expect(sun!.attributes('aria-label')).toBe("Yorug' rejim")
    expect(moon!.attributes('aria-label')).toBe("Qorong'i rejim")
    expect(sun!.attributes('aria-checked')).toBe('true')
    expect(moon!.attributes('aria-checked')).toBe('false')
    expect(w.get('[role="radiogroup"]').attributes('aria-label')).toBe('Mavzu')
  })

  it('switches to dark and back, saving the choice and moving the highlight', async () => {
    const w = mount(ThemeToggle)
    await w.get('[data-theme="dark"]').trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(w.get('[data-theme="dark"]').attributes('aria-checked')).toBe('true')
    expect(w.get('[data-theme="light"]').attributes('aria-checked')).toBe('false')
    expect(w.get('[data-theme="dark"]').classes()).toContain('bg-muted')

    await w.get('[data-theme="light"]').trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
  })

  it('keeps several toggles on the page in sync', async () => {
    const a = mount(ThemeToggle)
    const b = mount(ThemeToggle)
    await a.get('[data-theme="dark"]').trigger('click')
    expect(b.get('[data-theme="dark"]').attributes('aria-checked')).toBe('true')
  })
})
