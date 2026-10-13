import { describe, expect, it } from 'vitest'
import type { CurrentUser, Role } from '@/api/auth'
import { firstAllowedAdminPath, hasPermission } from './permissions'

const user = (role: Role, permissions: string[] = []): CurrentUser => ({
  id: '1', username: 'u', role, full_name: 'U', must_change_password: false, group_id: null, faculty_id: null, permissions,
})

describe('hasPermission', () => {
  it('lets admins do everything and others only what they hold', () => {
    expect(hasPermission(user('admin'), 'audit:view')).toBe(true)
    expect(hasPermission(user('little', ['users:view']), 'users:view')).toBe(true)
    expect(hasPermission(user('little', ['users:view']), 'users:update')).toBe(false)
    expect(hasPermission(user('teacher', ['users:view']), 'users:view')).toBe(false)
    expect(hasPermission(null, 'users:view')).toBe(false)
  })

  it('finds the first admin section a user may open, in sidebar order', () => {
    expect(firstAllowedAdminPath(user('little', ['audit:view', 'locations:view']))).toBe('/admin/locations')
    expect(firstAllowedAdminPath(user('little'))).toBeNull()
    expect(firstAllowedAdminPath(user('admin'))).toBe('/admin')
  })
})
