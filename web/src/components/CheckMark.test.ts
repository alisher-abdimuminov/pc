import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CheckMark from './CheckMark.vue'

describe('CheckMark', () => {
  it('shows a check icon when passed and an x icon when failed, with screen-reader text', () => {
    const ok = mount(CheckMark, { props: { ok: true, label: 'Yuz: Muvaffaqiyatli' } })
    expect(ok.find('svg').classes().join(' ')).toContain('lucide-check')
    expect(ok.find('svg').attributes('aria-hidden')).toBe('true')
    expect(ok.get('.sr-only').text()).toBe('Yuz: Muvaffaqiyatli')

    const bad = mount(CheckMark, { props: { ok: false, label: 'Yuz: Muvaffaqiyatsiz' } })
    expect(bad.find('svg').classes().join(' ')).toContain('lucide-x')
    expect(bad.get('.sr-only').text()).toBe('Yuz: Muvaffaqiyatsiz')
    expect(ok.text() + bad.text()).not.toMatch(/[✓✗]/)
  })
})
