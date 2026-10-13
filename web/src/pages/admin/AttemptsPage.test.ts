import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AdminAttempt } from '@/api/admin'
import AttemptsPage from './AttemptsPage.vue'

const api = vi.hoisted(() => ({ attempts: { list: vi.fn() }, groups: { list: vi.fn() }, users: { setActive: vi.fn() } }))
vi.mock('@/api/admin', () => api)

const attempt = (over: Partial<AdminAttempt> = {}): AdminAttempt => ({
  id: 'a1', attempted_at: '2026-10-12T05:00:00Z', date: '2026-10-12', shift: 1, step: 2,
  student_id: 's1', student_name: 'Test Talaba 1', student_username: 'student1', student_active: true, group_id: 'g1', group_name: 'TEST-101',
  face_verified: false, location_verified: true, liveness_verified: true, success: false,
  error: 'Yuz saqlangan namunaga mos kelmadi', face_distance: 0.8, face_threshold: 0.3, antispoof_score: 0.9,
  gps_accuracy: 12, latitude: 39.6505, longitude: 66.9505, ip: '10.0.0.5', has_selfie: true, ...over,
})

async function render(items: AdminAttempt[]) {
  api.attempts.list.mockResolvedValue({ items, total: items.length, page: 1, page_size: 20 })
  api.groups.list.mockResolvedValue({ items: [{ id: 'g1', name: 'TEST-101' }], total: 1, page: 1, page_size: 100 })
  const w = mount(AttemptsPage, {
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

describe('admin attempts page', () => {
  it('lists attempts with the three checks and the result', async () => {
    const w = await render([attempt(), attempt({ id: 'a2', success: true, face_verified: true, student_name: 'Boshqa Talaba' })])
    const rows = w.findAll('[data-testid="attempt-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('Test Talaba 1')
    expect(rows[0]!.text()).toContain('TEST-101')
    expect(rows[0]!.text()).toContain('1-smena')
    expect(rows[0]!.text()).toContain('Muvaffaqiyatsiz')
    expect(rows[0]!.find('[data-flag="face:false"]').exists()).toBe(true)
    expect(rows[0]!.find('[data-flag="loc:true"]').exists()).toBe(true)
    expect(rows[0]!.find('[data-flag="live:true"]').exists()).toBe(true)
    expect(rows[1]!.text()).toContain('Muvaffaqiyatli')
    expect(rows[0]!.text()).toContain('2-bosqich')
    expect(rows[0]!.text()).toContain('12.10.2026')
    expect(rows[0]!.text()).toContain('10:00') // 05:00Z is 10:00 in Tashkent
    expect(rows[1]!.find('[data-flag="face:true"]').exists()).toBe(true)
    expect(rows[0]!.find('a').attributes('href')).toBe('/admin/users/s1')
  })

  it('asks the API for the first page with no filters', async () => {
    await render([attempt()])
    expect(api.attempts.list).toHaveBeenCalledWith(expect.objectContaining({ page: 1, page_size: 20, search: '' }))
    const arg = api.attempts.list.mock.calls[0]![0]
    expect(arg.success).toBeUndefined()
    expect(arg.kind).toBeUndefined()
  })

  it('shows an empty state', async () => {
    const w = await render([])
    expect(w.text()).toContain("Ma'lumot topilmadi")
  })

  it('opens the details with scores, coordinates and the selfie', async () => {
    const w = await render([attempt()])
    await w.get('[data-testid="attempt-row"] td:nth-child(3)').trigger('click')
    await flushPromises()
    const dialog = document.body.textContent ?? ''
    expect(dialog).toContain('0.80 / 0.30')
    expect(dialog).toContain('39.650500, 66.950500')
    expect(dialog).toContain('10.0.0.5')
    expect(dialog).toContain('Yuz saqlangan namunaga mos kelmadi')
    expect(document.body.querySelector('[role="dialog"] img')?.getAttribute('src')).toBe('/api/files/selfies/a1')
    expect(document.body.querySelector('a[href="/admin/students/s1/attempts"]')).not.toBeNull()
  })

  it('says so when the selfie was already cleaned up', async () => {
    const w = await render([attempt({ has_selfie: false })])
    await w.get('[data-testid="attempt-row"] td:nth-child(3)').trigger('click')
    await flushPromises()
    expect(document.body.querySelector('[role="dialog"] img')).toBeNull()
    expect(document.body.textContent).toContain('Selfi saqlanmagan')
  })

  describe('blocking from the details dialog', () => {
    const open = async (over: Partial<AdminAttempt> = {}) => {
      const w = await render([attempt(over)])
      await w.get('[data-testid="attempt-row"] td:nth-child(3)').trigger('click')
      await flushPromises()
      return w
    }
    const q = (sel: string) => document.body.querySelector<HTMLElement>(sel)

    it('flags a spoofed attempt and blocks the student after a confirmation', async () => {
      api.users.setActive.mockResolvedValue({})
      await open({ liveness_verified: false })
      expect(q('[data-testid="spoof-warning"]')).not.toBeNull()
      q('[data-testid="block-button"]')!.click()
      await flushPromises()
      expect(api.users.setActive).not.toHaveBeenCalled() // needs the confirmation
      q('[data-testid="block-confirm-button"]')!.click()
      await flushPromises()
      expect(api.users.setActive).toHaveBeenCalledWith('s1', false)
      expect(q('[data-testid="blocked-note"]')).not.toBeNull()
      expect(q('[data-testid="block-button"]')).toBeNull()
    })

    it('can cancel the confirmation', async () => {
      await open()
      q('[data-testid="block-button"]')!.click()
      await flushPromises()
      ;[...document.body.querySelectorAll<HTMLElement>('[data-testid="block-confirm"] button')].at(-1)!.click()
      await flushPromises()
      expect(q('[data-testid="block-confirm"]')).toBeNull()
      expect(api.users.setActive).not.toHaveBeenCalled()
    })

    it('shows no button for an already blocked student', async () => {
      await open({ student_active: false })
      expect(q('[data-testid="blocked-note"]')).not.toBeNull()
      expect(q('[data-testid="block-button"]')).toBeNull()
    })

    it('shows the error when blocking fails', async () => {
      api.users.setActive.mockRejectedValue(new Error('boom'))
      await open()
      q('[data-testid="block-button"]')!.click()
      await flushPromises()
      q('[data-testid="block-confirm-button"]')!.click()
      await flushPromises()
      expect(q('[data-testid="block-section"] [role="alert"]')).not.toBeNull()
    })
  })

  it('shows the total, the range footer and an avatar fallback with the group under the name', async () => {
    api.attempts.list.mockResolvedValue({ items: [attempt()], total: 41, page: 1, page_size: 20 })
    api.groups.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100 })
    const w = await render([attempt()])
    expect(w.get('[data-testid="list-total"]').text()).toBe('1')
    expect(w.get('[data-testid="showing"]').text()).toBe("Ko'rsatilmoqda 1–1 / 1")
    expect(w.findAll('[data-testid="attempt-row"]')[0]!.text()).toContain('TT')
  })

  it('marks a spoof suspect row and colours the result', async () => {
    const w = await render([attempt({ liveness_verified: false }), attempt({ id: 'a2', success: true, face_verified: true })])
    const rows = w.findAll('[data-testid="attempt-row"]')
    expect(rows[0]!.find('[data-testid="spoof-icon"]').exists()).toBe(true)
    expect(rows[1]!.find('[data-testid="spoof-icon"]').exists()).toBe(false)
    expect(rows[1]!.get('[data-testid="result-badge"]').classes().join(' ')).toContain('emerald')
    expect(rows[0]!.get('[data-testid="result-badge"]').classes().join(' ')).toContain('rose')
  })

  it('opens the details when the row is clicked but not when the student link is', async () => {
    const w = await render([attempt()])
    await w.get('[data-testid="attempt-row"] a').trigger('click')
    await flushPromises()
    expect(document.body.querySelector('dl')).toBeNull()
    await w.get('[data-testid="attempt-row"] td:nth-child(3)').trigger('click')
    await flushPromises()
    expect(document.body.querySelector('dl')).not.toBeNull()
  })

  it('has the filters in a popover and the date range next to the search', async () => {
    const w = await render([attempt()])
    expect(w.find('[data-testid="filter-button"]').exists()).toBe(true)
    expect(w.find('[data-testid="date-range"]').exists()).toBe(true)
  })

  it('offers details and all attempts in the row menu', async () => {
    const w = await render([attempt()])
    await w.get('[data-testid="row-menu"]').trigger('keydown', { key: 'Enter' })
    await flushPromises()
    const items = [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')].map((i) => i.textContent!.trim())
    expect(items).toEqual(['Batafsil', 'Talabaning barcha urinishlari'])
  })
})
