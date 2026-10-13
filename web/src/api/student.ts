import { api } from './client'
import type { DayStatus, Totals } from './reports'

export interface TodayStep {
  step: number
  start: string
  end: string
  state: 'done' | 'pending' | 'missed'
  current: boolean
  attempts: number
  max_attempts: number
}
export interface Today {
  date: string
  server_time: string
  state: 'no_schedule' | 'before_start' | 'active' | 'finished'
  schedule: { id: string; name: string; location_name: string; location_url: string } | null
  shift: number
  shift_locked: boolean
  steps: TodayStep[]
  gps_max_accuracy_m: number
}
export interface AttemptResult {
  success: boolean
  shift: number
  step: number
  face_verified: boolean
  location_verified: boolean
  liveness_verified: boolean
  failures: string[]
  message: string
  attempts_used: number
  attempts_left: number
  steps_done: number
  /** too many face mismatches: the account was deactivated and the session ended */
  blocked: boolean
}
export interface Month {
  month: string
  from: string
  to: string
  dates: string[]
  cells: Record<string, { status: DayStatus; shift?: number; steps: number }>
  totals: Totals
}
export interface Submission {
  id: string
  submitted_at: string
  has_file: boolean
  grade: number | null
  feedback: string
  graded_at: string | null
}
export interface StudentAssignment {
  id: string
  title: string
  description: string
  has_file: boolean
  deadline: string
  overdue: boolean
  can_submit: boolean
  submission: Submission | null
}
export interface Profile {
  id: string
  username: string
  full_name: string
  short_name: string
  first_name: string
  second_name: string
  third_name: string
  birth_date: string
  gender: string
  phone: string
  level: string
  semester: string
  address: string
  country: string
  province: string
  district: string
  group_name: string | null
  faculty_name: string | null
  photo_url: string
}

const base = '/api/student'

export const student = {
  today: () => api<Today>(`${base}/today`),
  attempt: (image: Blob, pos: { latitude: number; longitude: number; accuracy: number }) => {
    const fd = new FormData()
    fd.set('image', image, 'selfie.jpg')
    fd.set('latitude', String(pos.latitude))
    fd.set('longitude', String(pos.longitude))
    fd.set('accuracy', String(pos.accuracy))
    return api<AttemptResult>(`${base}/attendance/attempt`, { method: 'POST', body: fd })
  },
  month: (month: string) => api<Month>(`${base}/attendance?month=${month}`),
  profile: () => api<Profile>(`${base}/profile`),
  assignments: () => api<{ server_time: string; items: StudentAssignment[] }>(`${base}/assignments`),
  submit: (id: string, file: File) => {
    const fd = new FormData()
    fd.set('file', file)
    return api<StudentAssignment>(`${base}/assignments/${id}/submission`, { method: 'PUT', body: fd })
  },
}
