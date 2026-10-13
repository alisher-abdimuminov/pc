import { describe, expect, it } from 'vitest'
import type { CurrentUser, Role } from '@/api/auth'
import { resolveRedirect } from './guard'

const user = (role: Role, must = false): CurrentUser => ({
  id: '1', username: 'u', role, full_name: 'U', must_change_password: must, group_id: null, faculty_id: null,
})
const route = (path: string, meta: { public?: boolean; roles?: Role[] } = {}) => ({ path, meta })

describe('resolveRedirect', () => {
  it('sends anonymous users to /login except on public routes', () => {
    expect(resolveRedirect(null, route('/admin', { roles: ['admin'] }))).toBe('/login')
    expect(resolveRedirect(null, route('/login', { public: true }))).toBeNull()
  })

  it('forces the password change page', () => {
    expect(resolveRedirect(user('admin', true), route('/admin'))).toBe('/change-password')
    expect(resolveRedirect(user('admin', true), route('/change-password'))).toBeNull()
  })

  it('redirects logged-in users away from login and the root to their home', () => {
    expect(resolveRedirect(user('teacher'), route('/login', { public: true }))).toBe('/teacher')
    expect(resolveRedirect(user('student'), route('/'))).toBe('/student')
    expect(resolveRedirect(user('admin'), route('/change-password'))).toBe('/admin')
  })

  it('keeps users inside their role area', () => {
    expect(resolveRedirect(user('teacher'), route('/admin', { roles: ['admin'] }))).toBe('/teacher')
    expect(resolveRedirect(user('admin'), route('/admin', { roles: ['admin'] }))).toBeNull()
  })
})
