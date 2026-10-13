import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StudentAttendancePage from './StudentAttendancePage.vue'

const api = vi.hoisted(() => ({ student: { month: vi.fn() } }))
vi.mock('@/api/student', () => api)

const dates = Array.from({ length: 31 }, (_, i) => `2026-10-${String(i + 1).padStart(2, '0')}`)

async function render() {
  api.student.month.mockResolvedValue({
    month: '2026-10', from: '2026-10-01', to: '2026-10-31', dates,
    cells: {
      '2026-10-05': { status: 'full', shift: 1, steps: 3 },
      '2026-10-06': { status: 'partial', shift: 2, steps: 1 },
      '2026-10-07': { status: 'absent', steps: 0 },
    },
    totals: { expected: 3, full: 1, partial: 1, absent: 1, percent: 33 },
  })
  const w = mount(StudentAttendancePage, {
    global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] },
  })
  await flushPromises()
  return w
}

beforeEach(() => vi.clearAllMocks())

describe('student attendance calendar marks', () => {
  it('marks days with Lucide icons for every status, never text symbols', async () => {
    const w = await render()
    const marks = w.findAll('[data-testid="day-mark"]')
    expect(marks).toHaveLength(3)
    expect(marks[0]!.find('svg').classes().join(' ')).toContain('lucide-check')
    expect(marks[1]!.find('svg').classes().join(' ')).toContain('lucide-contrast')
    expect(marks[1]!.text()).toBe('')
    expect(marks[2]!.find('svg').classes().join(' ')).toContain('lucide-x')
    expect(w.text()).not.toMatch(/[✓✗½‹›]/)
  })

  it('keeps each day readable for screen readers', async () => {
    const w = await render()
    const [full, partial, absent] = w.findAll('[data-testid="day-mark"]')
    expect(full!.attributes('aria-label')).toBe("2026-10-05: To'liq, 1-smena")
    expect(partial!.attributes('aria-label')).toBe('2026-10-06: Qisman, 2-smena')
    expect(absent!.attributes('aria-label')).toBe('2026-10-07: Kelmagan')
    expect(full!.find('svg').attributes('aria-hidden')).toBe('true')
  })

  it('uses the same icons in the legend and shows the totals', async () => {
    const w = await render()
    const legend = w.get('ul')
    expect(legend.findAll('svg')).toHaveLength(3)
    expect(w.get('dl').text()).toContain('3')
  })
})

describe('month navigation', () => {
  it('uses chevron icons on the previous and next buttons', async () => {
    const w = await render()
    const prev = w.get('button[aria-label="Oldingi oy"]')
    const next = w.get('button[aria-label="Keyingi oy"]')
    expect(prev.find('svg').classes().join(' ')).toContain('lucide-chevron-left')
    expect(next.find('svg').classes().join(' ')).toContain('lucide-chevron-right')
    expect(prev.text() + next.text()).toBe('')
    await prev.trigger('click')
    await flushPromises()
    expect(api.student.month).toHaveBeenLastCalledWith(expect.stringMatching(/^\d{4}-\d{2}$/))
  })
})
