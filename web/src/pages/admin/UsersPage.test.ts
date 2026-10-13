import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { User } from '@/api/admin'
import UsersPage from './UsersPage.vue'

const api = vi.hoisted(() => ({
  users: { list: vi.fn(), setActive: vi.fn(), regeneratePassword: vi.fn(), create: vi.fn() },
  faculties: { list: vi.fn() },
  groups: { list: vi.fn() },
}))
vi.mock('@/api/admin', () => api)

const user = (over: Partial<User> = {}): User => ({
  id: 'u1', username: 'ustoz1', role: 'teacher', is_active: true, must_change_password: false, last_login_at: '2026-10-12T05:00:00Z',
  hemis_login: '', hemis_synced_at: null, first_name: '', second_name: '', third_name: '', full_name: 'Karimov Toshpo\'lat', short_name: '',
  phone: '', gender: '', birth_date: '', level: '', semester: '', address: '', has_photo: false, face_enrolled: false,
  group_id: null, group_name: null, faculty_id: null, faculty_name: null, created_at: '2026-01-05T05:00:00Z', ...over,
})

async function render(items: User[], total = items.length, page = 1) {
  api.users.list.mockResolvedValue({ items, total, page, page_size: 20 })
  api.faculties.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100 })
  api.groups.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100 })
  const w = mount(UsersPage, {
    attachTo: document.body,
    global: {
      plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]],
      stubs: { RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] } },
    },
  })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('admin users page', () => {
  it('shows the total next to the title and a user row with avatar initials, role, status and dates', async () => {
    const w = await render([user(), user({ id: 'u2', full_name: 'Talaba Bir', username: '', role: 'student', group_name: 'TEST-101', is_active: false })], 453)
    expect(w.get('[data-testid="list-total"]').text()).toBe('453')
    const rows = w.findAll('[data-testid="user-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain("Karimov Toshpo'lat")
    expect(rows[0]!.text()).toContain('ustoz1') // subtitle
    expect(rows[0]!.text()).toContain('KT') // avatar fallback
    expect(rows[0]!.text()).toContain("O'qituvchi")
    expect(rows[0]!.get('[data-testid="status-badge"]').text()).toBe('Faol')
    expect(rows[1]!.text()).toContain('TEST-101') // the group is the subtitle for students
    expect(rows[1]!.get('[data-testid="status-badge"]').text()).toBe('Nofaol')
  })

  it('has an access switch that blocks and unblocks', async () => {
    api.users.setActive.mockResolvedValue({})
    const w = await render([user(), user({ id: 'u2', is_active: false })])
    const sw = w.findAll('[data-testid="access-switch"]')
    expect(sw[0]!.attributes('aria-checked')).toBe('true')
    expect(sw[1]!.attributes('aria-checked')).toBe('false')
    await sw[0]!.trigger('click')
    await flushPromises()
    expect(api.users.setActive).toHaveBeenCalledWith('u1', false)
    await sw[1]!.trigger('click')
    await flushPromises()
    expect(api.users.setActive).toHaveBeenCalledWith('u2', true)
  })

  it('shows the server error when the switch is refused', async () => {
    api.users.setActive.mockRejectedValue(new Error('boom'))
    const w = await render([user()])
    await w.get('[data-testid="access-switch"]').trigger('click')
    await flushPromises()
    expect(w.get('[role="alert"]').text().length).toBeGreaterThan(0)
  })

  it('selects rows and all rows', async () => {
    const w = await render([user(), user({ id: 'u2' })])
    const boxes = w.findAll('[data-testid="user-row"] [role="checkbox"]')
    await boxes[0]!.trigger('click')
    expect(boxes[0]!.attributes('aria-checked')).toBe('true')
    expect(w.get('[data-testid="select-all"]').attributes('aria-checked')).toBe('mixed')
    await w.get('[data-testid="select-all"]').trigger('click')
    await flushPromises()
    expect(w.findAll('[data-testid="user-row"] [role="checkbox"]').every((b) => b.attributes('aria-checked') === 'true')).toBe(true)
  })

  it('shows the visible range and moves between pages', async () => {
    const w = await render(Array.from({ length: 20 }, (_, i) => user({ id: `u${i}` })), 56)
    expect(w.get('[data-testid="showing"]').text()).toBe('Ko\'rsatilmoqda 1–20 / 56')
    const [prev, next] = w.findAll('button').filter((b) => ['Oldingi', 'Keyingi'].includes(b.text()))
    expect(prev!.attributes('disabled')).toBeDefined()
    await next!.trigger('click')
    await flushPromises()
    expect(api.users.list).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }))
  })

  it('marks HEMIS-linked teachers', async () => {
    const w = await render([user({ hemis_login: 't.k', hemis_synced_at: '2026-10-12T05:00:00Z' }), user({ id: 'u2' })])
    const rows = w.findAll('[data-testid="user-row"]')
    expect(rows[0]!.find('[data-testid="hemis-badge"]').exists()).toBe(true)
    expect(rows[1]!.find('[data-testid="hemis-badge"]').exists()).toBe(false)
  })
})
