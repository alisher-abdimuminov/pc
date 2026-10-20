import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ReviewSwitch from './ReviewSwitch.vue'

const api = vi.hoisted(() => ({ attempts: { review: vi.fn() }, teacher: { reviewAttempt: vi.fn() } }))
vi.mock('@/api/admin', () => ({ attempts: api.attempts }))
vi.mock('@/api/teacher', () => ({ teacher: api.teacher }))

const render = (props: { review: 'pending' | 'approved' | 'rejected'; as: 'admin' | 'teacher' }) =>
  mount(ReviewSwitch, {
    props: { attemptId: 'a1', ...props },
    global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] },
  })

describe('ReviewSwitch', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows a pending attempt in yellow with the switch off', () => {
    const w = render({ review: 'pending', as: 'teacher' })
    expect(w.get('[data-testid="review-badge"]').classes().join(' ')).toContain('yellow')
    expect(w.get('[data-testid="review-badge"]').text()).toBe('Tekshiruvda')
    expect(w.get('[data-testid="review-toggle"]').attributes('aria-checked')).toBe('false')
  })

  it('a teacher approves through the teacher endpoint', async () => {
    api.teacher.reviewAttempt.mockResolvedValue(undefined)
    const w = render({ review: 'pending', as: 'teacher' })
    await w.get('[data-testid="review-toggle"]').trigger('click')
    await flushPromises()
    expect(api.teacher.reviewAttempt).toHaveBeenCalledWith('a1', true)
    expect(api.attempts.review).not.toHaveBeenCalled()
  })

  it('an admin rejects an approved attempt through the admin endpoint', async () => {
    api.attempts.review.mockResolvedValue(undefined)
    const w = render({ review: 'approved', as: 'admin' })
    expect(w.get('[data-testid="review-toggle"]').attributes('aria-checked')).toBe('true')
    await w.get('[data-testid="review-toggle"]').trigger('click')
    await flushPromises()
    expect(api.attempts.review).toHaveBeenCalledWith('a1', false)
  })

  it('shows the server error', async () => {
    api.teacher.reviewAttempt.mockRejectedValue(new Error('boom'))
    const w = render({ review: 'pending', as: 'teacher' })
    await w.get('[data-testid="review-toggle"]').trigger('click')
    await flushPromises()
    expect(w.get('[role="alert"]').text()).not.toBe('')
  })
})
