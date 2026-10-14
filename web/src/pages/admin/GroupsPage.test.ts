import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/client'
import GroupsPage from './GroupsPage.vue'

const api = vi.hoisted(() => ({
  groups: { list: vi.fn(), sync: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn() },
  faculties: { list: vi.fn() },
  selectOptions: { faculties: vi.fn() },
  users: { list: vi.fn() },
}))
vi.mock('@/api/admin', () => api)

const page = (items: unknown[]) => ({ items, total: items.length, page: 1, page_size: 20 })

async function render() {
  api.groups.list.mockResolvedValue(page([{ id: 'g1', name: '101-guruh', faculty_name: null, teacher_name: null, student_count: 0 }]))
  api.selectOptions.faculties.mockResolvedValue([])
  api.users.list.mockResolvedValue(page([]))
  const w = mount(GroupsPage, {
    global: {
      plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]],
      stubs: { RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] } },
    },
  })
  await flushPromises()
  return w
}

beforeEach(() => vi.clearAllMocks())

describe('groups page: HEMIS sync', () => {
  it('runs the sync, reports the counts and reloads the list', async () => {
    api.groups.sync.mockResolvedValue({ total: 12, created: 5, updated: 2, unchanged: 5 })
    const w = await render()
    expect(api.groups.list).toHaveBeenCalledTimes(1)
    await w.get('[data-testid="sync-button"]').trigger('click')
    await flushPromises()
    expect(api.groups.sync).toHaveBeenCalledOnce()
    const box = w.get('[data-testid="sync-result"]').text()
    expect(box).toContain('Sinxronlash yakunlandi')
    expect(box).toContain('HEMIS dagi guruhlar: 12')
    expect(box).toContain('Yangi: 5')
    expect(box).toContain('Yangilangan: 2')
    expect(box).toContain("O'zgarishsiz: 5")
    expect(api.groups.list.mock.calls.length).toBeGreaterThan(1) // the table refreshed
  })

  it('shows the server message when HEMIS is down or not configured', async () => {
    api.groups.sync.mockRejectedValue(new ApiError(502, { code: 'hemis_unavailable', message: 'HEMIS tizimi vaqtincha ishlamayapti' }))
    const w = await render()
    await w.get('[data-testid="sync-button"]').trigger('click')
    await flushPromises()
    expect(w.get('[data-testid="sync-error"]').text()).toContain('HEMIS tizimi vaqtincha ishlamayapti')
    expect(w.find('[data-testid="sync-result"]').exists()).toBe(false)
  })

  it('disables the button while syncing so it cannot be pressed twice', async () => {
    let finish: (v: unknown) => void = () => undefined
    api.groups.sync.mockReturnValue(new Promise((r) => (finish = r)))
    const w = await render()
    await w.get('[data-testid="sync-button"]').trigger('click')
    await flushPromises()
    expect(w.get('[data-testid="sync-button"]').attributes('disabled')).toBeDefined()
    expect(w.get('[data-testid="sync-button"]').text()).toContain('Sinxronlanmoqda')
    await w.get('[data-testid="sync-button"]').trigger('click')
    expect(api.groups.sync).toHaveBeenCalledOnce()
    finish({ total: 0, created: 0, updated: 0, unchanged: 0 })
    await flushPromises()
    expect(w.get('[data-testid="sync-button"]').attributes('disabled')).toBeUndefined()
  })
})

describe('groups page layout', () => {
  it('shows the total, a row per group with its student count and a view link in the row menu', async () => {
    api.groups.list.mockResolvedValue({ items: [
      { id: 'g1', name: '101-guruh', faculty_name: 'Tillar', teacher_name: 'Ustoz Bir', student_count: 3, created_at: '2026-01-05T05:00:00Z' },
      { id: 'g2', name: '102-guruh', faculty_name: null, teacher_name: null, student_count: 0, created_at: '2026-01-05T05:00:00Z' },
    ], total: 40, page: 1, page_size: 20 })
    api.selectOptions.faculties.mockResolvedValue([])
    api.users.list.mockResolvedValue(page([]))
    const w = mount(GroupsPage, {
      attachTo: document.body,
      global: {
        plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]],
        stubs: { RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] } },
      },
    })
    await flushPromises()
    expect(w.get('[data-testid="list-total"]').text()).toBe('40')
    const rows = w.findAll('[data-testid="group-row"]')
    expect(rows[0]!.text()).toContain('3')
    expect(rows[0]!.text()).toContain('Tillar')
    expect(rows[0]!.text()).toContain('Ustoz Bir')
    expect(rows[1]!.text()).toContain('0')
    expect(w.get('[data-testid="showing"]').text()).toBe("Ko'rsatilmoqda 1–2 / 40")
    await rows[0]!.get('[data-testid="row-menu"]').trigger('keydown', { key: 'Enter' }) // opens the menu from the keyboard
    await flushPromises()
    const items = [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')].map((i) => i.textContent!.trim())
    expect(items).toEqual(["Ko'rish", 'Tahrirlash', "O'chirish"])
  })
})
