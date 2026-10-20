import { api } from './client'

export interface Page<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface ListQuery {
  page?: number
  page_size?: number
  search?: string
  [key: string]: string | number | boolean | undefined
}

function qs(q: ListQuery = {}): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(q)) {
    if (v !== undefined && v !== '' && v !== null) p.set(k, String(v))
  }
  const s = p.toString()
  return s ? `?${s}` : ''
}

const send = (method: string, body?: unknown): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: body === undefined ? undefined : JSON.stringify(body),
})

const base = '/api/admin'

// ---- types ----
export interface DayStats {
  date: string
  expected: number
  full: number
  partial: number
  absent: number
}
export interface Dashboard {
  students: number
  teachers: number
  groups: number
  today: DayStats
  last_7_days: DayStats[]
}
export interface Faculty {
  id: string
  name: string
  hemis_id: number | null
  student_count: number
  created_at: string
}
export interface Group {
  id: string
  name: string
  faculty_id: string | null
  faculty_name: string | null
  teacher_id: string | null
  teacher_name: string | null
  student_count: number
  created_at: string
}
export interface GroupDetail extends Group {
  students: User[]
}
export interface GroupInput {
  name: string
  faculty_id: string | null
  teacher_id: string | null
}
export interface User {
  id: string
  username: string
  role: 'admin' | 'little' | 'teacher' | 'student'
  is_active: boolean
  must_change_password: boolean
  last_login_at: string | null
  hemis_login: string
  hemis_synced_at: string | null
  permissions: string[]
  first_name: string
  second_name: string
  third_name: string
  full_name: string
  short_name: string
  phone: string
  gender: string
  birth_date: string
  level: string
  semester: string
  address: string
  has_photo: boolean
  face_enrolled: boolean
  group_id: string | null
  group_name: string | null
  faculty_id: string | null
  faculty_name: string | null
  created_at: string
}
export interface UserUpdate {
  full_name: string
  phone: string
  group_id: string | null
  faculty_id: string | null
  /** teachers only: "" unlinks, undefined leaves it unchanged */
  hemis_login?: string
}
export interface Sensitive {
  passport_pin: string
  passport_number: string
}
export interface Location {
  id: string
  name: string
  polygon: [number, number][]
  tolerance_meters: number
  location_url: string
  created_at: string
}
export interface LocationInput {
  name: string
  points: string[]
  tolerance_meters: number
  location_url: string
}
export interface Schedule {
  id: string
  name: string
  start_date: string
  end_date: string
  user_id: string | null
  user_name: string | null
  group_id: string | null
  group_name: string | null
  location_id: string | null
  location_name: string | null
  monday: boolean
  tuesday: boolean
  wednesday: boolean
  thursday: boolean
  friday: boolean
  saturday: boolean
  created_at: string
}
export type ScheduleInput = Omit<Schedule, 'id' | 'user_name' | 'group_name' | 'location_name' | 'created_at'>
export interface AuditEntry {
  id: string
  actor_id: string | null
  actor_role: string
  actor_username: string | null
  actor_name: string | null
  action: string
  entity_type: string
  entity_id: string | null
  changes: { before?: Record<string, unknown>; after?: Record<string, unknown> }
  ip: string
  user_agent: string
  created_at: string
}

// ---- calls ----
export const getDashboard = () => api<Dashboard>(`${base}/dashboard`)

function crud<T, I>(path: string) {
  return {
    list: (q?: ListQuery) => api<Page<T>>(`${base}/${path}${qs(q)}`),
    get: (id: string) => api<T>(`${base}/${path}/${id}`),
    create: (body: I) => api<T>(`${base}/${path}`, send('POST', body)),
    update: (id: string, body: I) => api<T>(`${base}/${path}/${id}`, send('PUT', body)),
    remove: (id: string) => api<void>(`${base}/${path}/${id}`, send('DELETE')),
  }
}

export interface SyncResult {
  total: number
  created: number
  updated: number
  unchanged: number
}

export const faculties = {
  ...crud<Faculty, { name: string }>('faculties'),
  sync: () => api<SyncResult>(`${base}/faculties/sync`, send('POST')),
}

export const groups = {
  ...crud<Group, GroupInput>('groups'),
  sync: () => api<SyncResult>(`${base}/groups/sync`, send('POST')),
  get: (id: string) => api<GroupDetail>(`${base}/groups/${id}`),
}
export const locations = crud<Location, LocationInput>('locations')
export const schedules = crud<Schedule, ScheduleInput>('schedules')

/** Unpaginated id + name lists for select boxes. */
export interface Option {
  id: string
  name: string
}
const options = (path: string) => () => api<{ items: Option[] }>(`${base}/options/${path}`).then((r) => r.items)
export const selectOptions = {
  faculties: options('faculties'),
  groups: options('groups'),
  locations: options('locations'),
  schedules: options('schedules'),
}

export const users = {
  list: (q?: ListQuery) => api<Page<User>>(`${base}/users${qs(q)}`),
  get: (id: string) => api<User>(`${base}/users/${id}`),
  create: (body: { username: string; role: string; full_name: string; hemis_login?: string; permissions?: string[] }) =>
    api<{ user: User; password: string }>(`${base}/users`, send('POST', body)),
  update: (id: string, body: UserUpdate) => api<User>(`${base}/users/${id}`, send('PUT', body)),
  setActive: (id: string, active: boolean) =>
    api<User>(`${base}/users/${id}/${active ? 'activate' : 'deactivate'}`, send('POST')),
  regeneratePassword: (id: string) =>
    api<{ password: string }>(`${base}/users/${id}/regenerate-password`, send('POST')),
  setPermissions: (id: string, permissions: string[]) => api<User>(`${base}/users/${id}/permissions`, send('PUT', { permissions })),
  sensitive: (id: string) => api<Sensitive>(`${base}/users/${id}/sensitive`),
  uploadPhoto: (id: string, file: File) => {
    const fd = new FormData()
    fd.set('image', file)
    return api<User>(`${base}/users/${id}/photo`, { method: 'PUT', body: fd })
  },
}

export interface AdminAttempt {
  id: string
  attempted_at: string
  date: string
  shift: number
  step: number
  student_id: string
  student_name: string
  student_username: string
  student_active: boolean
  group_id: string | null
  group_name: string | null
  face_verified: boolean
  location_verified: boolean
  liveness_verified: boolean
  success: boolean
  error: string
  face_distance: number | null
  face_threshold: number | null
  antispoof_score: number | null
  gps_accuracy: number
  latitude: number
  longitude: number
  ip: string
  has_selfie: boolean
  review: '' | 'pending' | 'approved' | 'rejected'
}

export const attempts = {
  list: (q?: ListQuery) => api<Page<AdminAttempt>>(`${base}/attempts${qs(q)}`),
  remove: (id: string) => api<void>(`${base}/attempts/${id}`, send('DELETE')),
  review: (id: string, approved: boolean) => api<void>(`${base}/attempts/${id}/review`, send('PUT', { approved })),
}

export const audit = {
  list: (q?: ListQuery) => api<Page<AuditEntry>>(`${base}/audit${qs(q)}`),
  get: (id: string) => api<AuditEntry>(`${base}/audit/${id}`),
}

export interface PermissionInfo {
  key: string
  /** everything this permission depends on (granted together with it) */
  requires: string[]
}
export const permissionCatalog = () => api<{ permissions: PermissionInfo[] }>(`${base}/permissions`).then((r) => r.permissions)
