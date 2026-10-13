import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AttemptsView from './AttemptsView.vue'

vi.mock('@/api/reports', () => ({
  fetchAttempts: vi.fn().mockResolvedValue({
    student: { id: 's1', full_name: 'Test Talaba 1', group_name: 'TEST-101' }, from: '2026-10-06', to: '2026-10-12',
    days: [{
      date: '2026-10-12', shift: 1,
      steps: [{
        step: 1, success: false,
        attempts: [{
          id: 'a1', attempted_at: '2026-10-12T05:00:00Z', face_verified: false, location_verified: true, liveness_verified: true,
          success: false, error: 'mos kelmadi', face_distance: 0.8, face_threshold: 0.3, antispoof_score: 0.9,
          gps_accuracy: 12, latitude: 1, longitude: 2, has_selfie: false,
        }],
      }],
    }],
  }),
}))

describe('attempt details marks', () => {
  it('shows face, location and liveness as check / x icons, readable by screen readers', async () => {
    const w = await (async () => {
      const m = mount(AttemptsView, {
        props: { url: '/api/x', cacheKey: 't' },
        global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] },
      })
      await flushPromises()
      return m
    })()
    const marks = w.findAll('tbody [data-ok]')
    expect(marks.map((m) => m.attributes('data-ok'))).toEqual(['false', 'true', 'true']) // face, location, liveness
    expect(marks[0]!.find('svg').classes().join(' ')).toContain('lucide-x')
    expect(marks[1]!.find('svg').classes().join(' ')).toContain('lucide-check')
    expect(marks[0]!.text()).toContain('Yuz: Muvaffaqiyatsiz')
    expect(marks[1]!.text()).toContain('Joylashuv: Muvaffaqiyatli')
    expect(w.text()).not.toMatch(/[✓✗]/)
  })
})
