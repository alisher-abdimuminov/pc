import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AttendanceReport from './AttendanceReport.vue'

vi.mock('@/api/reports', () => ({
  fetchReport: vi.fn().mockResolvedValue({
    from: '2026-10-05', to: '2026-10-07', dates: ['2026-10-05', '2026-10-06', '2026-10-07'],
    students: [{
      id: 's1', full_name: 'Test Talaba 1', group_id: 'g', group_name: 'TEST-101',
      cells: {
        '2026-10-05': { status: 'full', shift: 1, steps: 3 },
        '2026-10-06': { status: 'partial', shift: 2, steps: 1 },
        '2026-10-07': { status: 'absent', steps: 0 },
      },
      totals: { expected: 3, full: 1, partial: 1, absent: 1, percent: 33 },
    }],
  }),
}))

async function render() {
  const w = mount(AttendanceReport, {
    props: { url: '/api/x', exportUrl: '/api/x.xlsx', cacheKey: 't' },
    global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] },
  })
  await flushPromises()
  return w
}

describe('attendance report marks', () => {
  it('marks every status with a Lucide icon, no text symbols', async () => {
    const w = await render()
    const cells = w.findAll('td span[title]')
    expect(cells).toHaveLength(3)
    // full -> Check, partial -> half-filled circle, absent -> X
    expect(cells[0]!.find('svg').exists()).toBe(true)
    expect(cells[0]!.find('svg').classes().join(' ')).toContain('lucide-check')
    expect(cells[2]!.find('svg').classes().join(' ')).toContain('lucide-x')
    expect(cells[1]!.find('svg').classes().join(' ')).toContain('lucide-contrast')
    expect(cells[1]!.text()).toBe('1/3') // steps done, no symbol
    expect(cells[0]!.text()).toBe("To'liq")
    expect(cells[2]!.text()).toBe('Kelmagan')
    expect(w.text()).not.toContain('✓')
    expect(w.text()).not.toContain('✗')
  })

  it('keeps the meaning available to screen readers and on hover', async () => {
    const w = await render()
    const [full, partial, absent] = w.findAll('td span[title]')
    expect(full!.attributes('aria-label')).toBe("To'liq, 1-smena")
    expect(partial!.attributes('aria-label')).toBe('Qisman, 2-smena (1/3)')
    expect(absent!.attributes('aria-label')).toBe('Kelmagan')
    expect(full!.find('svg').attributes('aria-hidden')).toBe('true') // the label is on the cell, not the icon
  })

  it('shows the same icons in the legend', async () => {
    const w = await render()
    const legend = w.get('ul')
    expect(legend.findAll('svg')).toHaveLength(3)
    expect(legend.text()).not.toMatch(/[✓✗½]/)
  })

  it('lays the report out as a calendar grid like the design', async () => {
    const w = await render()
    const heads = w.findAll('thead th').map((h) => h.text())
    expect(heads[0]).toBe('Talaba')
    expect(heads[1]).toContain('Dushanba') // 2026-10-05
    expect(heads[1]).toContain('okt') // the month is named on the first column
    expect(heads[2]).toBe('Seshanba')
    // each day cell carries its day number above the status badge
    const cell = w.findAll('tbody tr')[0]!.findAll('td')[1]!
    expect(cell.text()).toBe("5 To'liq")
    // person cell: avatar fallback initials, name and group
    const person = w.findAll('tbody tr')[0]!.findAll('td')[0]!
    expect(person.text()).toContain('Test Talaba 1')
    expect(person.text()).toContain('TEST-101')
    expect(person.text()).toContain('TT')
  })

  it('hatches Sundays and days after today but not past working days', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-10-12T05:00:00Z')) // Monday
    const { fetchReport } = await import('@/api/reports')
    vi.mocked(fetchReport).mockResolvedValueOnce({
      from: '2026-10-10', to: '2026-10-14', dates: ['2026-10-10', '2026-10-11', '2026-10-12', '2026-10-13', '2026-10-14'],
      students: [{ id: 's', full_name: 'A B', group_id: null, group_name: null, cells: {}, totals: { expected: 0, full: 0, partial: 0, absent: 0, percent: 0 } }],
    })
    const w = await render()
    const hatched = w.findAll('tbody tr')[0]!.findAll('td').slice(1, 6).map((td) => td.attributes('data-hatched') === 'true')
    expect(hatched).toEqual([false, true, false, true, true]) // Sat, Sun, today (Mon), then the future
    vi.useRealTimers()
  })

  describe('filters', () => {
    const lastUrl = async () => {
      const { fetchReport } = await import('@/api/reports')
      return String(vi.mocked(fetchReport).mock.calls.at(-1)![0])
    }
    const week = (url: string) => new URLSearchParams(url.split('?')[1]!)

    it('has a segmented period control with the week selected', async () => {
      const w = await render()
      const modes = w.findAll('[role="radio"]')
      expect(modes.map((m) => m.text())).toEqual(['Kun', 'Hafta', 'Oy', 'Oraliq'])
      expect(modes.map((m) => m.attributes('aria-checked'))).toEqual(['false', 'true', 'false', 'false'])
    })

    it('switches the period and reloads with the matching range', async () => {
      const w = await render()
      await w.get('[data-testid="mode-day"]').trigger('click')
      await flushPromises()
      const day = week(await lastUrl())
      expect(day.get('from')).toBe(day.get('to'))
      await w.get('[data-testid="mode-month"]').trigger('click')
      await flushPromises()
      const month = week(await lastUrl())
      expect(month.get('from')!.endsWith('-01')).toBe(true)
      expect(w.get('[data-testid="range-label"]').text()).toMatch(/^01\.\d\d\.\d{4} – \d\d\.\d\d\.\d{4}$/)
    })

    it('steps back and forward with the arrows', async () => {
      const w = await render()
      const before = week(await lastUrl()).get('from')!
      await w.get('[data-testid="period-prev"]').trigger('click')
      await flushPromises()
      const prev = week(await lastUrl()).get('from')!
      expect(new Date(before).getTime() - new Date(prev).getTime()).toBe(7 * 86400000)
      await w.get('[data-testid="period-next"]').trigger('click')
      await flushPromises()
      expect(week(await lastUrl()).get('from')).toBe(before)
    })

    it('shows two pickers instead of the arrows for a custom range, and renders the extra filter slot', async () => {
      const w = mount(AttendanceReport, {
        props: { url: '/api/x', exportUrl: '/api/x.xlsx', cacheKey: 'slot' },
        slots: { filters: '<span data-testid="extra">Guruh</span>' },
        global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] },
      })
      await flushPromises()
      expect(w.find('[data-testid="extra"]').exists()).toBe(true)
      await w.get('[data-testid="mode-custom"]').trigger('click')
      expect(w.find('[data-testid="period-nav"]').exists()).toBe(false)
      expect(w.get('[data-testid="report-filters"]').findAll('button[id], button').length).toBeGreaterThan(5)
    })

    it('keeps the export link on the same range', async () => {
      const w = await render()
      const href = w.get('a[download]').attributes('href')!
      expect(href.startsWith('/api/x.xlsx?')).toBe(true)
      expect(href).toContain(`from=${week(await lastUrl()).get('from')}`)
    })
  })
})
