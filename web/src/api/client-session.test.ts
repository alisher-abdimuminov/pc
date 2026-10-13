import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, SESSION_EXPIRED_EVENT } from './client'

afterEach(() => vi.unstubAllGlobals())

const reply = (status: number) =>
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status, statusText: '', json: async () => ({ error: { code: 'x', message: 'm' } }) }))

describe('session expiry', () => {
  it('announces a 401 on a protected endpoint', async () => {
    const fired = vi.fn()
    window.addEventListener(SESSION_EXPIRED_EVENT, fired)
    reply(401)
    await api('/api/student/today').catch(() => undefined)
    window.removeEventListener(SESSION_EXPIRED_EVENT, fired)
    expect(fired).toHaveBeenCalledOnce()
  })

  it('does not announce failed logins or the initial /me probe', async () => {
    const fired = vi.fn()
    window.addEventListener(SESSION_EXPIRED_EVENT, fired)
    reply(401)
    await api('/api/auth/login').catch(() => undefined)
    await api('/api/auth/me').catch(() => undefined)
    reply(403)
    await api('/api/student/today').catch(() => undefined)
    window.removeEventListener(SESSION_EXPIRED_EVENT, fired)
    expect(fired).not.toHaveBeenCalled()
  })
})
