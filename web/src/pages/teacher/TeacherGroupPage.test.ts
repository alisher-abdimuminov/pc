import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TeacherGroupPage from './TeacherGroupPage.vue'

const api = vi.hoisted(() => ({ teacher: { group: vi.fn() } }))
vi.mock('@/api/teacher', () => api)
vi.mock('@/api/reports', () => ({
  fetchReport: vi.fn().mockResolvedValue({
    from: '2026-10-12', to: '2026-10-12', dates: ['2026-10-12'],
    students: [{ id: 's1', full_name: 'Test Talaba 1', group_id: 'g', group_name: 'TEST-101', cells: { '2026-10-12': { status: 'full', shift: 1, steps: 3 } }, totals: { expected: 1, full: 1, partial: 0, absent: 0, percent: 100 } }],
  }),
}))
vi.mock('vue-router', async (orig) => ({ ...(await orig<typeof import('vue-router')>()), useRoute: () => ({ params: { id: 'g1' } }) }))

async function render() {
  api.teacher.group.mockResolvedValue({
    id: 'g1', name: 'TEST-101', faculty_name: 'Fakultet', student_count: 1,
    today: { expected: 1, full: 1, partial: 0, absent: 0, attended_percent: 100 },
    students: [{ id: 's1', full_name: 'Test Talaba 1', status: 'full', shift: 1, steps: 3 }],
  })
  const w = mount(TeacherGroupPage, {
    global: {
      plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]],
      stubs: { RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] } },
    },
  })
  await flushPromises()
  return w
}

beforeEach(() => vi.clearAllMocks())

describe('teacher group page', () => {
  it("shows each student's attendance today without any link to attempt details", async () => {
    const w = await render()
    expect(w.text()).toContain('Test Talaba 1')
    expect(w.text()).toContain("To'liq")
    expect(w.findAll('a').map((a) => a.attributes('href')).filter((h) => h?.includes('students') || h?.includes('attempt'))).toEqual([])
  })

  it('keeps the attendance report, also without attempt links', async () => {
    const w = await render()
    const tabs = w.findAll('[role="tab"]')
    await tabs[1]!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Test Talaba 1')
    expect(w.find('td span[title^="To\'liq"]').exists()).toBe(true) // the day's mark (a check icon)
    expect(w.findAll('a').map((a) => a.attributes('href') ?? '').filter((h) => h.includes('students') || h.includes('attempt'))).toEqual([])
  })
})
