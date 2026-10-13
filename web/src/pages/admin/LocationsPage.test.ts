import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LocationsPage from './LocationsPage.vue'

const api = vi.hoisted(() => ({ locations: { list: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn() } }))
vi.mock('@/api/admin', () => api)
vi.mock('@/components/LocationMap.vue', () => ({ default: { template: '<div data-testid="map" />' } }))

const loc = (i: number, over = {}) => ({
  id: `l${i}`, name: `Joy ${i}`, polygon: [[1, 1], [1, 2], [2, 2]], tolerance_meters: 30, location_url: '', created_at: '2026-01-05T05:00:00Z', ...over,
})

async function render(items: unknown[], total = items.length) {
  api.locations.list.mockResolvedValue({ items, total, page: 1, page_size: 20 })
  const w = mount(LocationsPage, {
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

describe('locations page', () => {
  it('shows the total, rows with point count, tolerance and map link', async () => {
    const w = await render([loc(1, { location_url: 'https://maps.example/x', tolerance_meters: 50 }), loc(2)], 12)
    expect(w.get('[data-testid="list-total"]').text()).toBe('12')
    const rows = w.findAll('[data-testid="location-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('Joy 1')
    expect(rows[0]!.text()).toContain('3 ta nuqta')
    expect(rows[0]!.text()).toContain('50 m')
    expect(rows[0]!.get('a').attributes('href')).toBe('https://maps.example/x')
    expect(rows[0]!.get('a').attributes('rel')).toContain('noopener')
    expect(rows[1]!.find('a').exists()).toBe(false)
    expect(w.get('[data-testid="showing"]').text()).toBe("Ko'rsatilmoqda 1–2 / 12")
  })

  it('opens the map from the name and from the row menu', async () => {
    const w = await render([loc(1)])
    await w.get('[data-testid="location-name"]').trigger('click')
    await flushPromises()
    expect(document.body.querySelector('[data-testid="map"]')).not.toBeNull()
  })

  it('offers view, edit and delete in the row menu', async () => {
    const w = await render([loc(1)])
    await w.get('[data-testid="row-menu"]').trigger('keydown', { key: 'Enter' })
    await flushPromises()
    const items = [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')].map((i) => i.textContent!.trim())
    expect(items).toEqual(["Xaritada ko'rish", 'Tahrirlash', "O'chirish"])
  })

  it('selects rows', async () => {
    const w = await render([loc(1), loc(2)])
    await w.findAll('[data-testid="location-row"] [role="checkbox"]')[1]!.trigger('click')
    expect(w.get('[data-testid="select-all"]').attributes('aria-checked')).toBe('mixed')
  })
})
