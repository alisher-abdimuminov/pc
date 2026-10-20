import { api } from './client'
import type { ListQuery, Page } from './admin'

export interface TodayStats {
  expected: number
  full: number
  partial: number
  absent: number
  attended_percent: number
}
export interface TeacherGroup {
  id: string
  name: string
  faculty_name: string | null
  student_count: number
  today: TodayStats
}
export interface GroupStudentToday {
  id: string
  full_name: string
  status: '' | 'full' | 'partial' | 'absent'
  shift: number
  steps: number
  /** steps waiting for a manual review */
  pending: number
}
export interface TeacherGroupDetail extends TeacherGroup {
  students: GroupStudentToday[]
}
export interface Assignment {
  id: string
  group_id: string
  group_name: string
  title: string
  description: string
  has_file: boolean
  deadline: string
  submission_count: number
  student_count: number
  created_at: string
}
export interface RosterEntry {
  student_id: string
  full_name: string
  submission_id: string | null
  submitted_at: string | null
  has_file: boolean
  grade: number | null
  feedback: string
  graded_at: string | null
}
export interface SubmissionsView {
  assignment: Assignment
  submitted: RosterEntry[]
  missing: RosterEntry[]
}

const qs = (q: ListQuery = {}) => {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(q)) if (v !== undefined && v !== '') p.set(k, String(v))
  const s = p.toString()
  return s ? `?${s}` : ''
}

const base = '/api/teacher'

export const teacher = {
  reviewAttempt: (id: string, approved: boolean) =>
    api<void>(`${base}/attempts/${id}/review`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ approved }) }),
  groups: () => api<{ items: TeacherGroup[] }>(`${base}/groups`).then((r) => r.items),
  group: (id: string) => api<TeacherGroupDetail>(`${base}/groups/${id}`),
  assignments: (q?: ListQuery) => api<Page<Assignment>>(`${base}/assignments${qs(q)}`),
  assignment: (id: string) => api<Assignment>(`${base}/assignments/${id}`),
  saveAssignment: (id: string | null, form: FormData) =>
    api<Assignment>(id ? `${base}/assignments/${id}` : `${base}/assignments`, { method: id ? 'PUT' : 'POST', body: form }),
  deleteAssignment: (id: string) => api<void>(`${base}/assignments/${id}`, { method: 'DELETE' }),
  submissions: (id: string) => api<SubmissionsView>(`${base}/assignments/${id}/submissions`),
  grade: (submissionId: string, grade: number, feedback: string) =>
    api<RosterEntry>(`${base}/submissions/${submissionId}/grade`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ grade, feedback }),
    }),
}
