import { describe, expect, it } from 'vitest'
import type { CurrentUser, Role } from '@/api/auth'
import { resolveRedirect } from './guard'

const user = (role: Role, must = false): CurrentUser => ({
  id: '1', username: 'u', role, full_name: 'U', must_change_password: must, group_id: null, faculty_id: null, permissions: [],
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

describe('little admin', () => {
  const little = (permissions: string[]): CurrentUser => ({ ...user('little'), permissions })
  const adminPage = (path: string, permission?: string) => ({ path, meta: { roles: ['admin', 'little'] as Role[], permission } })

  it('works in the admin area', () => {
    expect(resolveRedirect(little(['users:view']), route('/login'))).toBe('/admin')
    expect(resolveRedirect(little(['users:view']), route('/teacher', { roles: ['teacher'] }))).toBe('/admin')
  })

  it('opens a page it was granted and bounces from one it was not to the first allowed section', () => {
    const u = little(['groups:view', 'faculties:view'])
    expect(resolveRedirect(u, adminPage('/admin/groups', 'groups:view'))).toBeNull()
    expect(resolveRedirect(u, adminPage('/admin/audit', 'audit:view'))).toBe('/admin/faculties')
    expect(resolveRedirect(u, adminPage('/admin', 'dashboard:view'))).toBe('/admin/faculties')
  })

  it('goes to the dashboard when that is granted', () => {
    expect(resolveRedirect(little(['dashboard:view', 'audit:view']), adminPage('/admin/users', 'users:view'))).toBe('/admin')
  })

  it('lands on a no-access page when nothing was granted, without looping', () => {
    const u = little([])
    expect(resolveRedirect(u, adminPage('/admin', 'dashboard:view'))).toBe('/no-access')
    expect(resolveRedirect(u, route('/login'))).toBe('/no-access')
    expect(resolveRedirect(u, route('/no-access'))).toBeNull()
  })

  it('keeps no-access for those who have something', () => {
    expect(resolveRedirect(little(['audit:view']), route('/no-access'))).toBe('/admin')
  })

  it('admins pass every permission check', () => {
    expect(resolveRedirect(user('admin'), adminPage('/admin/audit', 'audit:view'))).toBeNull()
  })
})
