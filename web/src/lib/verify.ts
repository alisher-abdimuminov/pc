import type { Today } from '@/api/student'
import { uz } from '@/locales/uz'

/** Browsers expose camera and geolocation only in secure contexts (HTTPS or localhost). */
export function secureContextProblem(isSecureContext: boolean): string | null {
  return isSecureContext ? null : uz.verify.insecure
}

/** Maps GeolocationPositionError.code (1 denied, 2 unavailable, 3 timeout). */
export function geoErrorMessage(code: number): string {
  switch (code) {
    case 1:
      return uz.verify.geo.denied
    case 2:
      return uz.verify.geo.unavailable
    case 3:
      return uz.verify.geo.timeout
    default:
      return uz.verify.geo.unknown
  }
}

/** Maps a getUserMedia DOMException name. */
export function cameraErrorMessage(name: string): string {
  switch (name) {
    case 'NotAllowedError':
    case 'SecurityError':
    case 'PermissionDeniedError':
      return uz.verify.cam.denied
    case 'NotFoundError':
    case 'DevicesNotFoundError':
    case 'OverconstrainedError':
      return uz.verify.cam.notFound
    case 'NotReadableError':
    case 'TrackStartError':
    case 'AbortError':
      return uz.verify.cam.busy
    default:
      return uz.verify.cam.unknown
  }
}

export interface Guidance {
  code: string
  title: string
  hint: string
}

/** What failed and what the student should do about it, in the order the server reported. */
export function failureGuidance(codes: string[]): Guidance[] {
  const table = uz.verify.failures as Record<string, readonly [string, string]>
  return codes.map((code) => {
    const entry = table[code]
    return { code, title: entry?.[0] ?? code, hint: entry?.[1] ?? '' }
  })
}

export function accuracyLevel(accuracyM: number, limitM: number): 'good' | 'poor' {
  return accuracyM <= limitM ? 'good' : 'poor'
}

/** Size that fits inside max x max while keeping the aspect ratio; never upscales. */
export function scaleToFit(width: number, height: number, max: number): { width: number; height: number } {
  const k = Math.min(1, max / Math.max(width, height))
  return { width: Math.round(width * k), height: Math.round(height * k) }
}

/** Whether the "confirm" button should be offered on the home screen. */
export function canVerify(today: Today | undefined): { ok: boolean; step: number } {
  if (!today || today.state !== 'active' || today.shift_locked) return { ok: false, step: 0 }
  const cur = today.steps.find((s) => s.current)
  if (!cur || cur.state === 'done' || cur.attempts >= cur.max_attempts) return { ok: false, step: 0 }
  return { ok: true, step: cur.step }
}

export const MAX_CAPTURE_PX = 1024
