import { api } from './client'

export type DayStatus = 'full' | 'partial' | 'absent' | ''

export interface ReportCell {
  status: DayStatus
  shift?: number
  steps: number
  /** steps accepted without a face check (service was busy) that nobody has confirmed yet */
  pending?: number
}
export interface Totals {
  expected: number
  full: number
  partial: number
  absent: number
  percent: number
}
export interface ReportStudent {
  id: string
  full_name: string
  group_id: string | null
  group_name: string | null
  cells: Record<string, ReportCell>
  totals: Totals
}
export interface Report {
  from: string
  to: string
  dates: string[]
  students: ReportStudent[]
}

export interface Attempt {
  id: string
  attempted_at: string
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
  has_selfie: boolean
  review: Review
}
export type Review = '' | 'pending' | 'approved' | 'rejected'
export interface StepAttempts {
  step: number
  success: boolean
  review: Review
  attempts: Attempt[]
}
export interface AttemptDay {
  date: string
  shift: number
  steps: StepAttempts[]
}
export interface StudentAttempts {
  student: { id: string; full_name: string; group_name: string | null }
  from: string
  to: string
  days: AttemptDay[]
}

export const fetchReport = (url: string) => api<Report>(url)
export const fetchAttempts = (url: string) => api<StudentAttempts>(url)
