import { api } from './client'

export type Role = 'admin' | 'little' | 'teacher' | 'student'

export interface CurrentUser {
  id: string
  username: string
  role: Role
  full_name: string
  must_change_password: boolean
  group_id: string | null
  faculty_id: string | null
  /** what a "little" admin may use; empty for everyone else (admins can do everything) */
  permissions: string[]
}

const json = (body: unknown): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
})

export const login = (username: string, password: string) =>
  api<{ user: CurrentUser }>('/api/auth/login', json({ username, password })).then((r) => r.user)

export const me = () => api<{ user: CurrentUser }>('/api/auth/me').then((r) => r.user)

export const logout = () => api<void>('/api/auth/logout', { method: 'POST' })

export const changePassword = (old_password: string, new_password: string) =>
  api<{ user: CurrentUser }>('/api/auth/change-password', json({ old_password, new_password })).then((r) => r.user)
