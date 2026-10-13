import type { CurrentUser, Role } from '@/api/auth'
import { firstAllowedAdminPath, hasPermission } from '@/lib/permissions'

/** A little admin works in the admin area; the sections it sees depend on its permissions. */
export const homeFor = (role: Role) => (role === 'little' ? '/admin' : `/${role}`)

export interface GuardTarget {
  path: string
  meta: { public?: boolean; roles?: Role[]; permission?: string }
}

/** Returns the path to redirect to, or null when navigation may proceed. */
export function resolveRedirect(user: CurrentUser | null, to: GuardTarget): string | null {
  if (!user) return to.meta.public ? null : '/login'
  if (user.must_change_password) return to.path === '/change-password' ? null : '/change-password'
  if (to.path === '/login' || to.path === '/change-password' || to.path === '/') {
    return user.role === 'little' && !firstAllowedAdminPath(user) ? '/no-access' : homeFor(user.role)
  }
  if (to.path === '/no-access') return user.role === 'little' && !firstAllowedAdminPath(user) ? null : homeFor(user.role)
  if (to.meta.roles && !to.meta.roles.includes(user.role)) return homeFor(user.role)
  if (to.meta.permission && !hasPermission(user, to.meta.permission)) {
    // land on the first section the account may open (itself, if it has none, to avoid a loop)
    const first = firstAllowedAdminPath(user)
    if (!first) return '/no-access'
    return first !== to.path ? first : null
  }
  return null
}
