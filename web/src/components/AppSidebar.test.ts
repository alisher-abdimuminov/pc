import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { House, Users } from '@lucide/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import RoleLayout from './RoleLayout.vue'
import { useSessionStore } from '@/stores/session'

const logoutApi = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))
vi.mock('@/api/auth', () => ({ me: vi.fn(), login: vi.fn(), logout: logoutApi, changePassword: vi.fn() }))

const Page = { template: '<p data-testid="page">page</p>' }
const links = [
  { to: '/admin', label: 'Boshqaruv paneli', icon: House, exact: true },
  { to: '/admin/groups', label: 'Guruhlar', icon: Users },
]

async function mountAt(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/admin', component: Page },
      { path: '/admin/groups', component: Page },
      { path: '/admin/groups/:id', component: Page },
      { path: '/login', component: Page },
    ],
  })
  await router.push(path)
  const pinia = createPinia()
  setActivePinia(pinia)
  useSessionStore().setUser({
    id: '1', username: 'admin', role: 'admin', full_name: 'Ali Valiyev', must_change_password: false, group_id: null, faculty_id: null, permissions: [],
  })
  const w = mount(RoleLayout, { props: { title: 'Administrator', links }, global: { plugins: [router, pinia] } })
  await flushPromises()
  return { w, router }
}

beforeEach(() => vi.clearAllMocks())

describe('role layout with sidebar', () => {
  it('lists every link and marks the current page', async () => {
    const { w } = await mountAt('/admin/groups')
    const hrefs = w.findAll('a').map((a) => a.attributes('href'))
    expect(hrefs).toEqual(expect.arrayContaining(['/admin', '/admin/groups']))
    const current = w.findAll('a[aria-current="page"]')
    expect(current).toHaveLength(1)
    expect(current[0]!.text()).toBe('Guruhlar')
    expect(w.get('[data-testid="page-label"]').text()).toBe('Guruhlar')
  })

  it('keeps the section home exact but matches nested routes by prefix', async () => {
    const home = await mountAt('/admin')
    expect(home.w.get('a[aria-current="page"]').text()).toBe('Boshqaruv paneli')
    const nested = await mountAt('/admin/groups/123')
    expect(nested.w.get('a[aria-current="page"]').text()).toBe('Guruhlar') // detail page keeps its section highlighted
    expect(nested.w.get('[data-testid="page-label"]').text()).toBe('Guruhlar')
  })

  it('renders the routed page inside the layout', async () => {
    const { w } = await mountAt('/admin')
    expect(w.find('[data-testid="page"]').exists()).toBe(true)
  })

  it('shows who is signed in and logs out to the login page', async () => {
    const { w, router } = await mountAt('/admin')
    expect(w.get('[data-testid="sidebar-user"]').text()).toBe('Ali Valiyev')
    await w.get('[data-testid="sidebar-logout"]').trigger('click')
    await flushPromises()
    expect(logoutApi).toHaveBeenCalledOnce()
    expect(router.currentRoute.value.path).toBe('/login')
    expect(useSessionStore().user).toBeNull()
  })

  it('collapses on phones for phone-first roles (bottom navigation takes over)', async () => {
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/admin', component: Page }] })
    await router.push('/admin')
    const pinia = createPinia()
    const w = mount(RoleLayout, { props: { title: 'T', links, bottomNav: true }, global: { plugins: [router, pinia] } })
    await flushPromises()
    expect(w.get('header').classes()).toContain('max-md:hidden')
  })
})
