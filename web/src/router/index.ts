import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import type { Role } from '@/api/auth'
import { useSessionStore } from '@/stores/session'
import { resolveRedirect } from './guard'

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    roles?: Role[]
    permission?: string
  }
}

const layoutRoute = (
  role: Role,
  layout: () => Promise<unknown>,
  children: RouteRecordRaw[] = [],
  home: () => Promise<unknown> = () => import('@/pages/DashboardPage.vue'),
  { roles = [role], homePermission }: { roles?: Role[]; homePermission?: string } = {},
) => ({
  path: `/${role}`,
  component: layout,
  meta: { roles },
  children: [{ path: '', name: `${role}-home`, component: home, meta: { permission: homePermission } }, ...children],
})

const adminChildren: RouteRecordRaw[] = [
  { path: 'faculties', meta: { permission: 'faculties:view' }, component: () => import('@/pages/admin/FacultiesPage.vue') },
  { path: 'groups', meta: { permission: 'groups:view' }, component: () => import('@/pages/admin/GroupsPage.vue') },
  { path: 'groups/:id', meta: { permission: 'groups:view' }, component: () => import('@/pages/admin/GroupDetailPage.vue') },
  { path: 'users', meta: { permission: 'users:view' }, component: () => import('@/pages/admin/UsersPage.vue') },
  { path: 'users/:id', meta: { permission: 'users:view' }, component: () => import('@/pages/admin/UserDetailPage.vue') },
  { path: 'locations', meta: { permission: 'locations:view' }, component: () => import('@/pages/admin/LocationsPage.vue') },
  { path: 'schedules', meta: { permission: 'schedules:view' }, component: () => import('@/pages/admin/SchedulesPage.vue') },
  { path: 'audit', meta: { permission: 'audit:view' }, component: () => import('@/pages/admin/AuditPage.vue') },
  { path: 'attempts', meta: { permission: 'attempts:view' }, component: () => import('@/pages/admin/AttemptsPage.vue') },
  { path: 'reports', meta: { permission: 'reports:view' }, component: () => import('@/pages/admin/ReportsPage.vue') },
  { path: 'students/:id/attempts', meta: { permission: 'attempts:view' }, component: () => import('@/pages/admin/StudentAttemptsPage.vue') },
]

const studentChildren: RouteRecordRaw[] = [
  { path: 'verify', component: () => import('@/pages/student/VerifyPage.vue') },
  { path: 'attendance', component: () => import('@/pages/student/StudentAttendancePage.vue') },
  { path: 'assignments', component: () => import('@/pages/student/StudentAssignmentsPage.vue') },
  { path: 'profile', component: () => import('@/pages/student/StudentProfilePage.vue') },
]

const teacherChildren: RouteRecordRaw[] = [
  { path: 'groups/:id', component: () => import('@/pages/teacher/TeacherGroupPage.vue') },
  { path: 'students/:id', component: () => import('@/pages/teacher/TeacherStudentAttemptsPage.vue') },
  { path: 'assignments', component: () => import('@/pages/teacher/AssignmentsPage.vue') },
  { path: 'assignments/:id', component: () => import('@/pages/teacher/AssignmentDetailPage.vue') },
]

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('@/pages/NotFoundPage.vue') }, // always redirected by the guard
    { path: '/login', name: 'login', component: () => import('@/pages/LoginPage.vue'), meta: { public: true } },
    { path: '/no-access', name: 'no-access', component: () => import('@/pages/NoAccessPage.vue') },
    { path: '/change-password', name: 'change-password', component: () => import('@/pages/ChangePasswordPage.vue') },
    layoutRoute('admin', () => import('@/layouts/AdminLayout.vue'), adminChildren, () => import('@/pages/admin/AdminDashboardPage.vue'), { roles: ['admin', 'little'], homePermission: 'dashboard:view' }),
    layoutRoute('teacher', () => import('@/layouts/TeacherLayout.vue'), teacherChildren, () => import('@/pages/teacher/TeacherGroupsPage.vue')),
    layoutRoute('student', () => import('@/layouts/StudentLayout.vue'), studentChildren, () => import('@/pages/student/StudentHomePage.vue')),
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/pages/NotFoundPage.vue'), meta: { public: true } },
  ],
})

router.beforeEach(async (to) => {
  const session = useSessionStore()
  if (!session.loaded) await session.load()
  const target = resolveRedirect(session.user, to)
  return target && target !== to.path ? target : true
})
