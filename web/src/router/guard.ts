import type { CurrentUser, Role } from '@/api/auth'

export const homeFor = (role: Role) => `/${role}`

export interface GuardTarget {
  path: string
  meta: { public?: boolean; roles?: Role[] }
}

/** Returns the path to redirect to, or null when navigation may proceed. */
export function resolveRedirect(user: CurrentUser | null, to: GuardTarget): string | null {
  if (!user) return to.meta.public ? null : '/login'
  if (user.must_change_password) return to.path === '/change-password' ? null : '/change-password'
  if (to.path === '/login' || to.path === '/change-password' || to.path === '/') return homeFor(user.role)
  if (to.meta.roles && !to.meta.roles.includes(user.role)) return homeFor(user.role)
  return null
}
