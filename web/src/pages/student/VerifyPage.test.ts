import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { AttemptResult, Today } from '@/api/student'
import { ApiError } from '@/api/client'
import { useSessionStore } from '@/stores/session'
import VerifyPage from './VerifyPage.vue'

const api = vi.hoisted(() => ({ today: vi.fn(), attempt: vi.fn() }))
vi.mock('@/api/student', () => ({ student: api }))

// The camera itself is covered by FaceCapture.test.ts; here it is a stub that records its props.
let capture: ((b: Blob) => void) | null = null
const seen: { active?: boolean; blockedMessage?: string; busy?: boolean } = {}
const FaceCaptureStub = defineComponent({
  props: { active: Boolean, blockedMessage: String, busy: Boolean },
  emits: ['capture'],
  setup(props, { emit }) {
    capture = (b) => emit('capture', b)
    return () => {
      Object.assign(seen, { active: props.active, blockedMessage: props.blockedMessage, busy: props.busy })
      return h('div', { 'data-testid': 'camera', 'data-active': String(props.active), 'data-hold': props.blockedMessage ?? '' })
    }
  },
})

const today = (over: Partial<Today> = {}): Today => ({
  date: '2026-10-12', server_time: '2026-10-12T05:00:00Z', state: 'active', schedule: null, shift: 1, shift_locked: false,
  gps_max_accuracy_m: 100,
  steps: [{ step: 1, start: '08:00', end: '10:00', state: 'pending', current: true, attempts: 0, max_attempts: 5 }],
  ...over,
})
const result = (over: Partial<AttemptResult> = {}): AttemptResult => ({
  success: true, shift: 1, step: 1, face_verified: true, location_verified: true, liveness_verified: true, failures: [],
  message: 'Tasdiqlandi', attempts_used: 1, attempts_left: 4, steps_done: 1, blocked: false, ...over,
})

function geo(accuracy: number | null) {
  Object.defineProperty(window, 'isSecureContext', { configurable: true, value: true })
  Object.defineProperty(navigator, 'geolocation', {
    configurable: true,
    value: {
      watchPosition: (ok: PositionCallback) => {
        if (accuracy !== null) ok({ coords: { latitude: 39.65, longitude: 66.95, accuracy } } as GeolocationPosition)
        return 1
      },
      clearWatch: vi.fn(),
    },
  })
}

async function render(t = today()) {
  api.today.mockResolvedValue(t)
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/student/verify', component: VerifyPage }, { path: '/login', component: { template: '<p>login</p>' } }, { path: '/student', component: { template: '<p>home</p>' } }] })
  await router.push('/student/verify')
  const pinia = createPinia()
  const w = mount(VerifyPage, {
    global: {
      plugins: [router, pinia, [VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]],
      stubs: { FaceCapture: FaceCaptureStub },
    },
  })
  await flushPromises()
  return { w, router, pinia }
}

beforeEach(() => {
  vi.clearAllMocks()
  capture = null
  geo(10)
})
afterEach(() => vi.restoreAllMocks())

describe('verify page with automatic capture', () => {
  it('runs the camera when a step is open and the location is good', async () => {
    const { w } = await render()
    expect(w.get('[data-testid="camera"]').attributes('data-active')).toBe('true')
    expect(w.get('[data-testid="camera"]').attributes('data-hold')).toBe('')
    expect(w.text()).toContain('±10 m')
  })

  it('holds the countdown until the location is known', async () => {
    geo(null)
    const { w } = await render()
    expect(w.get('[data-testid="camera"]').attributes('data-hold')).toContain('Joylashuv aniqlanishini kuting')
  })

  it('holds the countdown while GPS accuracy is poor, without using up an attempt', async () => {
    geo(250)
    const { w } = await render()
    expect(w.get('[data-testid="camera"]').attributes('data-hold')).toContain('GPS aniqligi past')
    expect(api.attempt).not.toHaveBeenCalled()
  })

  it('does not start the camera when no step can be confirmed', async () => {
    const { w } = await render(today({ state: 'finished' }))
    expect(w.find('[data-testid="camera"]').exists()).toBe(false)
    expect(w.text()).toContain('Hozir tasdiqlash mumkin emas')
  })

  it('sends the photo that FaceCapture took together with the location', async () => {
    api.attempt.mockResolvedValue(result())
    const { w } = await render()
    const photo = new Blob(['jpeg'], { type: 'image/jpeg' })
    capture!(photo)
    await flushPromises()
    expect(api.attempt).toHaveBeenCalledWith(photo, { latitude: 39.65, longitude: 66.95, accuracy: 10 })
    expect(w.text()).toContain('Tasdiqlandi')
    expect(w.text()).toContain('1/3')
    expect(w.find('[data-testid="camera"]').exists()).toBe(false) // result screen replaces the camera
    expect(w.get('[role="status"] svg').classes().join(' ')).toContain('lucide-check') // an icon, not a ✓ character
    expect(w.text()).not.toMatch(/[✓✗]/)
  })

  it('explains what failed and starts a fresh try on demand', async () => {
    api.attempt.mockResolvedValue(result({ success: false, face_verified: false, failures: ['face', 'location'], attempts_left: 3 }))
    const { w } = await render()
    capture!(new Blob(['x']))
    await flushPromises()
    expect(w.text()).toContain('Tasdiqlanmadi')
    expect(w.text()).toContain('Yuz saqlangan namunaga mos kelmadi')
    expect(w.text()).toContain('Siz amaliyot joyida emassiz')
    expect(w.get('[role="status"] svg').classes().join(' ')).toContain('lucide-x')
    expect(w.findAll('li svg').filter((i) => i.classes().join(' ').includes('lucide-x'))).toHaveLength(2) // one per failed check
    expect(w.text()).not.toMatch(/[✓✗]/)
    expect(w.text()).toContain('3')
    await w.findAll('button').find((b) => b.text() === 'Qayta urinish')!.trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="camera"]').exists()).toBe(true) // camera is back for the next try
  })

  it('shows the server error when the attempt is rejected', async () => {
    api.attempt.mockRejectedValue(new ApiError(409, { code: 'step_done', message: 'Bu bosqich allaqachon tasdiqlangan' }))
    const { w } = await render()
    capture!(new Blob(['x']))
    await flushPromises()
    expect(w.get('[role="alert"]').text()).toContain('allaqachon tasdiqlangan')
  })

  it('logs the student out and shows the warning when the account was just blocked', async () => {
    api.attempt.mockResolvedValue(result({ success: false, face_verified: false, failures: ['face'], blocked: true, message: 'Hisobingiz faol emas. Tizim adminlariga murojaat qiling.' }))
    const { w, router, pinia } = await render()
    useSessionStore(pinia).setUser({ id: '1', username: 's', role: 'student', full_name: 'S', must_change_password: false, group_id: null, faculty_id: null })
    capture!(new Blob(['x']))
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/login')
    expect(router.currentRoute.value.query.reason).toBe('inactive')
    expect(useSessionStore(pinia).user).toBeNull()
    expect(w.exists()).toBe(true)
  })
})
