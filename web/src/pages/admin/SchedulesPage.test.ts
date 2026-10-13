import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Schedule } from '@/api/admin'
import SchedulesPage from './SchedulesPage.vue'

const api = vi.hoisted(() => ({
  schedules: { list: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn() },
  groups: { list: vi.fn() },
  locations: { list: vi.fn() },
  users: { list: vi.fn() },
}))
vi.mock('@/api/admin', () => api)

const sched = (over: Partial<Schedule> = {}): Schedule => ({
  id: 's1', name: 'Amaliyot', start_date: '2020-01-01', end_date: '2099-12-31', user_id: null, user_name: null,
  group_id: 'g', group_name: 'TEST-101', location_id: 'l', location_name: 'Narpay',
  monday: true, tuesday: true, wednesday: true, thursday: false, friday: false, saturday: false, created_at: '2026-01-05T05:00:00Z', ...over,
})
const page = (items: unknown[], total = items.length) => ({ items, total, page: 1, page_size: 20 })

async function render(items: Schedule[], total = items.length) {
  api.schedules.list.mockResolvedValue(page(items, total))
  api.groups.list.mockResolvedValue(page([]))
  api.locations.list.mockResolvedValue(page([]))
  api.users.list.mockResolvedValue(page([]))
  const w = mount(SchedulesPage, {
    attachTo: document.body,
    global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] },
  })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('schedules page', () => {
  it('shows the total, the schedule name with its period, the target and the location', async () => {
    const w = await render([sched(), sched({ id: 's2', name: '', group_id: null, group_name: null, user_id: 'u', user_name: 'Talaba Bir' })], 9)
    expect(w.get('[data-testid="list-total"]').text()).toBe('9')
    const rows = w.findAll('[data-testid="schedule-row"]')
    expect(rows[0]!.text()).toContain('Amaliyot')
    expect(rows[0]!.get('[data-testid="period"]').text()).toBe('01.01.2020 – 31.12.2099')
    expect(rows[0]!.text()).toContain('TEST-101')
    expect(rows[0]!.text()).toContain('Guruh')
    expect(rows[0]!.text()).toContain('Narpay')
    expect(rows[1]!.text()).toContain('Talaba Bir')
    expect(rows[1]!.text()).toContain('Talaba')
    expect(w.get('[data-testid="showing"]').text()).toBe("Ko'rsatilmoqda 1–2 / 9")
  })

  it('highlights the working weekdays as chips', async () => {
    const w = await render([sched()])
    const chips = w.findAll('[data-testid="weekdays"] span')
    expect(chips).toHaveLength(6)
    expect(chips.map((c) => c.attributes('data-on') === 'true')).toEqual([true, true, true, false, false, false])
    expect(chips[0]!.attributes('title')).toBe('Dushanba')
  })

  it('tells whether a schedule is running, upcoming or over', async () => {
    const w = await render([
      sched(),
      sched({ id: 'u', start_date: '2098-01-01', end_date: '2099-01-01' }),
      sched({ id: 'e', start_date: '2020-01-01', end_date: '2020-02-01' }),
    ])
    expect(w.findAll('[data-testid="state-badge"]').map((b) => b.text())).toEqual(['Faol', 'Kutilmoqda', 'Tugagan'])
  })

  it('offers edit and delete in the row menu', async () => {
    const w = await render([sched()])
    await w.get('[data-testid="row-menu"]').trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect([...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')].map((i) => i.textContent!.trim())).toEqual(['Tahrirlash', "O'chirish"])
  })
})
