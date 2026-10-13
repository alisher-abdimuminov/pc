export interface ApiErrorBody {
  code: string
  message: string
  fields?: Record<string, string>
}

export class ApiError extends Error {
  status: number
  body: ApiErrorBody
  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.status = status
    this.body = body
  }
}

/** Fired when a request that needed a session gets 401 (expired, logged out elsewhere, or blocked). */
export const SESSION_EXPIRED_EVENT = 'session-expired'

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, { credentials: 'same-origin', ...init })
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const body: ApiErrorBody = data?.error ?? { code: 'unknown', message: res.statusText }
    if (res.status === 401 && !path.startsWith('/api/auth/')) window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT))
    throw new ApiError(res.status, body)
  }
  return data as T
}
