import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Today, TodayStep } from '@/api/student'
import StudentHomePage from './StudentHomePage.vue'

const api = vi.hoisted(() => ({ today: vi.fn(), profile: vi.fn() }))
vi.mock('@/api/student', () => ({ student: api }))

const step = (n: number, start: string, end: string, over: Partial<TodayStep> = {}): TodayStep => ({
  step: n, start, end, state: 'pending', current: false, attempts: 0, max_attempts: 5, ...over,
})

const today = (over: Partial<Today> = {}): Today => ({
  date: '2026-10-12',
  server_time: '2026-10-12T05:00:00Z', // 10:00 in Tashkent
  state: 'active',
  schedule: { id: 's', name: '', location_name: 'Samarqand viloyat Narpay tumani', location_url: '' },
  shift: 1,
  shift_locked: false,
  gps_max_accuracy_m: 100,
  steps: [
    step(1, '08:00', '10:00', { state: 'missed' }),
    step(2, '10:00', '12:00', { current: true }),
    step(3, '12:00', '13:00'),
  ],
  ...over,
})

function geo(impl: (ok: PositionCallback, err: PositionErrorCallback) => void) {
  Object.defineProperty(navigator, 'geolocation', { configurable: true, value: { getCurrentPosition: impl, watchPosition: vi.fn(), clearWatch: vi.fn() } })
  Object.defineProperty(window, 'isSecureContext', { configurable: true, value: true })
}

async function render(t: Today) {
  api.today.mockResolvedValue(t)
  api.profile.mockResolvedValue({
    full_name: 'SHOMURODOVA DILNOZA', group_name: 'TILLAR_2024_04', faculty_name: 'Tillar', photo_url: '/api/files/avatars/x',
  })
  const w = mount(StudentHomePage, {
    global: {
      plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]],
      stubs: { RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] } },
    },
  })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] })
  vi.setSystemTime(new Date('2026-10-12T05:00:00Z'))
  geo((ok) => ok({ coords: { accuracy: 12 } } as GeolocationPosition))
})
afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
})

describe('student home', () => {
  it('shows the profile card with name, group and faculty', async () => {
    const w = await render(today())
    expect(w.get('[data-testid="student-name"]').text()).toBe('SHOMURODOVA DILNOZA')
    expect(w.text()).toContain('TILLAR_2024_04 • Tillar')
    expect(w.find('img').attributes('src')).toBe('/api/files/avatars/x')
  })

  it('renders the day as a timeline: missed, open and upcoming steps in order', async () => {
    const w = await render(today())
    const cards = w.findAll('[data-look]')
    expect(cards.map((c) => c.attributes('data-look'))).toEqual(['missed', 'open', 'upcoming'])
    expect(cards[0]!.text()).toContain("O'tkazildi")
    expect(cards[0]!.text()).toContain('Vaqti tugagan')
    expect(cards[1]!.text()).toContain('Ochiq')
    expect(cards[1]!.attributes('aria-current')).toBe('step')
    expect(cards[2]!.text()).toContain('12:00 da ochiladi')
    expect(w.get('[data-testid="progress"]').text()).toBe('0 / 3')
    expect(w.get('[data-testid="shift-line"]').text()).toBe('1-smena • Samarqand viloyat Narpay tumani')
  })

  it('sizes cards by duration and puts the now-marker at the current time', async () => {
    const w = await render(today())
    const heights = w.findAll('[data-look]').map((c) => parseInt((c.element as HTMLElement).style.height))
    expect(heights).toEqual([360 - 8, 360 - 8, 180 - 8]) // 2h, 2h, 1h at 3px/min
    const marker = w.get('[data-testid="now-marker"]')
    expect(marker.text()).toBe('10:00')
    expect((marker.element as HTMLElement).style.top).toBe('360px') // two hours after 08:00
  })

  it('offers the confirm button only on the open step', async () => {
    const w = await render(today())
    const open = w.get('[data-look="open"]')
    expect(open.find('[data-testid="verify-button"]').exists()).toBe(true)
    expect(w.findAll('[data-testid="verify-button"]')).toHaveLength(1)
    expect(open.get('a').attributes('href')).toBe('/student/verify')
  })

  it('counts confirmed steps and marks them done', async () => {
    const w = await render(today({ steps: [step(1, '08:00', '10:00', { state: 'done', attempts: 1 }), step(2, '10:00', '12:00', { current: true, attempts: 2 }), step(3, '12:00', '13:00')] }))
    expect(w.get('[data-testid="progress"]').text()).toBe('1 / 3')
    expect(w.get('[data-look="done"]').text()).toContain('Tasdiqlandi')
    expect(w.get('[data-look="done"]').find('svg').classes().join(' ')).toContain('lucide-check') // "Tasdiqlangan" tag with an icon
    expect(w.get('[data-look="done"]').text()).not.toMatch(/[✓✗]/)
    expect(w.get('[data-look="open"]').text()).toContain('2/5')
  })

  it('has no confirm button when the shift is locked or attempts are used up', async () => {
    const locked = await render(today({ shift_locked: true, steps: [step(1, '08:00', '10:00', { state: 'done' }), step(2, '10:00', '12:00'), step(3, '12:00', '13:00')] }))
    expect(locked.find('[data-testid="verify-button"]').exists()).toBe(false)
    expect(locked.text()).toContain('boshqa smenada')
    const spent = await render(today({ steps: [step(1, '08:00', '10:00', { state: 'missed' }), step(2, '10:00', '12:00', { current: true, attempts: 5 }), step(3, '12:00', '13:00')] }))
    expect(spent.find('[data-testid="verify-button"]').exists()).toBe(false)
  })

  it('shows a message instead of a timeline when there is no schedule', async () => {
    const w = await render(today({ state: 'no_schedule', schedule: null, shift: 0, steps: [] }))
    expect(w.find('[data-testid="timeline"]').exists()).toBe(false)
    expect(w.get('[data-testid="banner"]').text()).toContain("jadvali yo'q")
  })

  it('hides the now-marker outside the shift window', async () => {
    vi.setSystemTime(new Date('2026-10-12T14:00:00Z')) // 19:00 Tashkent
    const w = await render(today({ state: 'finished', server_time: '2026-10-12T14:00:00Z' }))
    expect(w.find('[data-testid="now-marker"]').exists()).toBe(false)
    expect(w.get('[data-testid="banner"]').text()).toContain('yakunlandi')
  })
})

describe('location status card', () => {
  it('shows the fix accuracy when located', async () => {
    const w = await render(today())
    const card = w.get('[data-testid="location-card"]')
    expect(card.text()).toContain('Joylashuv aniqlandi')
    expect(card.text()).toContain('±12 m')
  })

  it('flags a poor fix', async () => {
    geo((ok) => ok({ coords: { accuracy: 250 } } as GeolocationPosition))
    const w = await render(today())
    expect(w.get('[data-testid="location-card"]').text()).toContain('aniqlik past')
  })

  it('explains a denied permission and lets the user retry', async () => {
    const getCurrentPosition = vi.fn((_ok: PositionCallback, err: PositionErrorCallback) => err({ code: 1 } as GeolocationPositionError))
    geo(getCurrentPosition)
    const w = await render(today())
    const card = w.get('[data-testid="location-card"]')
    expect(card.text()).toContain('Joylashuv aniqlanmadi')
    expect(card.text()).toContain('Lokatsiyaga ruxsat berilmadi')
    await card.get('button').trigger('click')
    expect(getCurrentPosition).toHaveBeenCalledTimes(2)
  })

  it('reports an insecure context', async () => {
    geo((ok) => ok({ coords: { accuracy: 5 } } as GeolocationPosition))
    Object.defineProperty(window, 'isSecureContext', { configurable: true, value: false })
    const w = await render(today())
    expect(w.get('[data-testid="location-card"]').text()).toContain('HTTPS')
  })
})
