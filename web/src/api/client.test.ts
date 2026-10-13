import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError } from './client'

afterEach(() => vi.unstubAllGlobals())

describe('api', () => {
  it('returns parsed JSON on success', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ a: 1 }) }))
    expect(await api('/x')).toEqual({ a: 1 })
  })

  it('throws ApiError with the backend error body', async () => {
    const body = { error: { code: 'validation_error', message: 'xato', fields: { name: 'kerak' } } }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 422, statusText: '', json: async () => body }))
    const err = (await api('/x').catch((e: unknown) => e)) as ApiError
    expect(err).toBeInstanceOf(ApiError)
    expect(err.body.fields).toEqual({ name: 'kerak' })
  })
})
