import { getActivePinia } from 'pinia'
import type { CurrentUser } from '@/api/auth'
import { useSessionStore } from '@/stores/session'

/** Admins can do everything; a "little" admin only what it was granted; everyone else nothing. */
export function hasPermission(user: CurrentUser | null | undefined, permission: string): boolean {
  if (!user) return false
  if (user.role === 'admin') return true
  return user.role === 'little' && user.permissions.includes(permission)
}

/**
 * The admin sections, in sidebar order, with the permission each needs. The same list drives the sidebar
 * and the route guard, so a link is shown exactly when its page can be opened.
 */
export const ADMIN_SECTIONS = [
  { path: '/admin', key: 'dashboard', permission: 'dashboard:view' },
  { path: '/admin/faculties', key: 'faculties', permission: 'faculties:view' },
  { path: '/admin/groups', key: 'groups', permission: 'groups:view' },
  { path: '/admin/users', key: 'users', permission: 'users:view' },
  { path: '/admin/locations', key: 'locations', permission: 'locations:view' },
  { path: '/admin/schedules', key: 'schedules', permission: 'schedules:view' },
  { path: '/admin/reports', key: 'reports', permission: 'reports:view' },
  { path: '/admin/attempts', key: 'attempts', permission: 'attempts:view' },
  { path: '/admin/audit', key: 'audit', permission: 'audit:view' },
] as const

export function firstAllowedAdminPath(user: CurrentUser): string | null {
  return ADMIN_SECTIONS.find((s) => hasPermission(user, s.permission))?.path ?? null
}

/**
 * `can('users:update')` for templates. This only hides controls: the server enforces every permission.
 * Without an active Pinia (isolated component tests) everything is shown.
 */
export function useIsAdmin(): () => boolean {
  if (!getActivePinia()) return () => true
  const session = useSessionStore()
  return () => session.user?.role === 'admin'
}

export function useCan(): (permission: string) => boolean {
  if (!getActivePinia()) return () => true
  const session = useSessionStore()
  return (permission) => hasPermission(session.user, permission)
}
