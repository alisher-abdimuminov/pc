import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/client'
import FacultiesPage from './FacultiesPage.vue'

const api = vi.hoisted(() => ({ faculties: { list: vi.fn(), sync: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn() } }))
vi.mock('@/api/admin', () => api)

const fac = (i: number, over = {}) => ({ id: `f${i}`, name: `Fakultet ${i}`, hemis_id: null, created_at: '2026-01-05T05:00:00Z', ...over })

async function render(items: unknown[], total = items.length) {
  api.faculties.list.mockResolvedValue({ items, total, page: 1, page_size: 20 })
  const w = mount(FacultiesPage, {
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

describe('faculties page', () => {
  it('shows the total, rows with an icon tile, HEMIS id and the range footer', async () => {
    const w = await render([fac(1, { hemis_id: 55 }), fac(2)], 30)
    expect(w.get('[data-testid="list-total"]').text()).toBe('30')
    const rows = w.findAll('[data-testid="faculty-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('Fakultet 1')
    expect(rows[0]!.text()).toContain('HEMIS ID: 55')
    expect(rows[1]!.text()).not.toContain('HEMIS ID')
    expect(rows[0]!.find('svg.lucide-building-2').exists()).toBe(true)
    expect(w.get('[data-testid="showing"]').text()).toBe("Ko'rsatilmoqda 1–2 / 30")
  })

  it('selects rows', async () => {
    const w = await render([fac(1), fac(2)])
    await w.findAll('[data-testid="faculty-row"] [role="checkbox"]')[0]!.trigger('click')
    expect(w.get('[data-testid="select-all"]').attributes('aria-checked')).toBe('mixed')
  })

  it('offers edit and delete in the row menu and deletes after confirmation', async () => {
    api.faculties.remove.mockResolvedValue(undefined)
    const w = await render([fac(1)])
    await w.get('[data-testid="row-menu"]').trigger('keydown', { key: 'Enter' }) // opens the menu from the keyboard
    await flushPromises()
    const items = [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')]
    expect(items.map((i) => i.textContent!.trim())).toEqual(['Tahrirlash', "O'chirish"])
    items[1]!.click()
    await flushPromises()
    const confirm = [...document.body.querySelectorAll<HTMLElement>('[role="dialog"] button')].find((b) => b.textContent!.trim() === "O'chirish")!
    confirm.click()
    await flushPromises()
    expect(api.faculties.remove).toHaveBeenCalledWith('f1')
  })
})

describe('faculties page: HEMIS sync', () => {
  it('runs the sync, reports the counts and reloads the list', async () => {
    api.faculties.sync.mockResolvedValue({ total: 9, created: 4, updated: 1, unchanged: 4 })
    const w = await render([fac(1)])
    expect(w.get('[data-testid="sync-button"]').text()).toBe('HEMISdan sinxronlash')
    await w.get('[data-testid="sync-button"]').trigger('click')
    await flushPromises()
    expect(api.faculties.sync).toHaveBeenCalledOnce()
    const box = w.get('[data-testid="sync-result"]').text()
    expect(box).toContain('HEMISdagi fakultetlar: 9')
    expect(box).toContain('Yangi: 4')
    expect(box).toContain('Yangilangan: 1')
    expect(api.faculties.list.mock.calls.length).toBeGreaterThan(1)
  })

  it('shows the server message when HEMIS fails', async () => {
    api.faculties.sync.mockRejectedValue(new ApiError(502, { code: 'hemis_unavailable', message: 'HEMIS tizimi vaqtincha ishlamayapti' }))
    const w = await render([fac(1)])
    await w.get('[data-testid="sync-button"]').trigger('click')
    await flushPromises()
    expect(w.get('[data-testid="sync-error"]').text()).toContain('HEMIS tizimi')
    expect(w.find('[data-testid="sync-result"]').exists()).toBe(false)
  })
})
