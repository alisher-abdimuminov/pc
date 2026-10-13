import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import LoginPage from './LoginPage.vue'

const OAUTH_CODES = [
  'hemis_state', 'hemis_denied', 'hemis_token', 'hemis_profile', 'hemis_not_teacher',
  'hemis_inactive', 'hemis_not_registered', 'hemis_unavailable',
]

async function mountAt(url: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/login', component: LoginPage }] })
  await router.push(url)
  const w = mount(LoginPage, { global: { plugins: [router, createPinia()] } })
  await flushPromises()
  return Object.assign(w, { router })
}

describe('login page', () => {
  it('warns a blocked student after they are logged out', async () => {
    const w = await mountAt('/login?reason=inactive')
    expect(w.get('[data-testid="inactive-warning"]').text()).toBe('Hisobingiz faol emas. Tizim adminlariga murojaat qiling.')
  })

  it('shows no warning on a normal visit', async () => {
    const w = await mountAt('/login')
    expect(w.find('[data-testid="inactive-warning"]').exists()).toBe(false)
  })

  it('toggles the password between hidden and visible', async () => {
    const w = await mountAt('/login')
    const input = w.get('#password')
    const toggle = w.get('[data-testid="toggle-password"]')
    expect(input.attributes('type')).toBe('password')
    expect(toggle.attributes('type')).toBe('button') // must not submit the form
    expect(toggle.attributes('aria-label')).toBe("Parolni ko'rsatish")
    await toggle.trigger('click')
    expect(input.attributes('type')).toBe('text')
    expect(toggle.attributes('aria-label')).toBe('Parolni yashirish')
    await toggle.trigger('click')
    expect(input.attributes('type')).toBe('password')
  })

  describe('HEMIS sign-in for teachers', () => {
    it('is a plain link to the server start endpoint, not a fetch', async () => {
      const w = await mountAt('/login')
      const btn = w.get('[data-testid="hemis-teacher"]')
      expect(btn.element.tagName).toBe('A')
      expect(btn.attributes('href')).toBe('/api/auth/hemis/teacher/start')
      expect(btn.text()).toBe("HEMIS orqali kirish (O'qituvchi)")
      expect(w.find('[data-testid="oauth-error"]').exists()).toBe(false)
    })

    it('shows a loading state after the click', async () => {
      const w = await mountAt('/login')
      const btn = w.get('[data-testid="hemis-teacher"]')
      btn.element.addEventListener('click', (e) => e.preventDefault()) // jsdom cannot navigate
      await btn.trigger('click')
      expect(btn.text()).toContain("HEMIS ga o'tilmoqda")
      expect(btn.attributes('aria-disabled')).toBe('true')
    })

    it.each(OAUTH_CODES)('explains ?error=%s in Uzbek and clears the query', async (code) => {
      const w = await mountAt(`/login?error=${code}`)
      const msg = w.get('[data-testid="oauth-error"]').text()
      expect(msg.length).toBeGreaterThan(10)
      expect(msg).not.toContain(code)
      expect(w.router.currentRoute.value.query.error).toBeUndefined()
    })

    it('has a distinct message for every code', async () => {
      const texts = new Set<string>()
      for (const code of OAUTH_CODES.filter((c) => c !== 'hemis_inactive')) {
        texts.add((await mountAt(`/login?error=${code}`)).get('[data-testid="oauth-error"]').text())
      }
      expect(texts.size).toBe(OAUTH_CODES.length - 1)
    })

    it('reuses the standard blocked-account text for hemis_inactive', async () => {
      const w = await mountAt('/login?error=hemis_inactive')
      expect(w.get('[data-testid="oauth-error"]').text()).toBe('Hisobingiz faol emas. Tizim adminlariga murojaat qiling.')
    })

    it('falls back to a generic message for an unknown code and keeps other params', async () => {
      const w = await mountAt('/login?error=whatever&reason=inactive')
      expect(w.get('[data-testid="oauth-error"]').text().length).toBeGreaterThan(5)
      expect(w.router.currentRoute.value.query).toEqual({ reason: 'inactive' })
    })
  })
})
