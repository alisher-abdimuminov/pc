import { uz } from '@/locales/uz'

/** Normalised (0..1) landmark as returned by MediaPipe. */
export interface Landmark {
  x: number
  y: number
}

/** How long the face must be held steady before the photo is taken. */
export const STABLE_MS = 3000

/** Framing and pose limits, in normalised image units and degrees. */
export const LIMITS = {
  centerX: [0.35, 0.65],
  centerY: [0.3, 0.7],
  minWidth: 0.2,
  minHeight: 0.25,
  maxWidth: 0.65,
  maxHeight: 0.75,
  poseDeg: 7,
} as const

export interface Pose {
  pitch: number
  yaw: number
  roll: number
}

const deg = (r: number) => (r * 180) / Math.PI

/**
 * Head rotation in degrees from MediaPipe's facial transformation matrix
 * (4x4, column-major, rotation scaled by the face size). Euler order YXZ.
 */
export function headPose(m: ArrayLike<number>): Pose {
  const r = [0, 1, 2].map((i) => [0, 1, 2].map((j) => m[j * 4 + i] ?? 0)) // r[row][col]
  for (let j = 0; j < 3; j++) {
    const n = Math.hypot(r[0]![j]!, r[1]![j]!, r[2]![j]!) || 1 // drop the uniform scale
    for (let i = 0; i < 3; i++) r[i]![j]! /= n
  }
  const s = Math.max(-1, Math.min(1, -r[1]![2]!))
  const pitch = Math.asin(s)
  if (Math.abs(s) < 0.9999) {
    return { pitch: deg(pitch), yaw: deg(Math.atan2(r[0]![2]!, r[2]![2]!)), roll: deg(Math.atan2(r[1]![0]!, r[1]![1]!)) }
  }
  return { pitch: deg(pitch), yaw: deg(Math.atan2(-r[2]![0]!, r[0]![0]!)), roll: 0 } // gimbal lock
}

export type GuideKey = keyof typeof uz.verify.guide

export interface Verdict {
  ok: boolean
  /** message to show: guidance when not ok, "hold still" when ok */
  key: GuideKey
}

/** Decides whether the current frame is a good photo: one face, centred, right size, looking straight. */
export function evaluateFace(faces: Landmark[][], matrix?: ArrayLike<number>): Verdict {
  if (!faces.length) return { ok: false, key: 'noFace' }
  if (faces.length > 1) return { ok: false, key: 'multiple' }
  const points = faces[0]
  if (!points?.length) return { ok: false, key: 'noFace' }

  let x1 = 1, y1 = 1, x2 = 0, y2 = 0
  for (const p of points) {
    x1 = Math.min(x1, p.x)
    y1 = Math.min(y1, p.y)
    x2 = Math.max(x2, p.x)
    y2 = Math.max(y2, p.y)
  }
  const w = x2 - x1
  const h = y2 - y1
  const cx = (x1 + x2) / 2
  const cy = (y1 + y2) / 2

  if (cx <= LIMITS.centerX[0] || cx >= LIMITS.centerX[1] || cy <= LIMITS.centerY[0] || cy >= LIMITS.centerY[1]) {
    return { ok: false, key: 'center' }
  }
  if (w < LIMITS.minWidth || h < LIMITS.minHeight) return { ok: false, key: 'closer' }
  if (w > LIMITS.maxWidth || h > LIMITS.maxHeight) return { ok: false, key: 'farther' }
  if (!matrix) return { ok: false, key: 'noPose' }

  const { pitch, yaw, roll } = headPose(matrix)
  if (Math.abs(pitch) > LIMITS.poseDeg || Math.abs(yaw) > LIMITS.poseDeg || Math.abs(roll) > LIMITS.poseDeg) {
    return { ok: false, key: 'straight' }
  }
  return { ok: true, key: 'hold' }
}

/** Tracks how long the frames have been continuously good. */
export class StableTimer {
  private since: number | null = null

  /** Feed every frame; returns progress 0..1 (1 = held for STABLE_MS). */
  update(ok: boolean, now: number): number {
    if (!ok) {
      this.since = null
      return 0
    }
    this.since ??= now
    return Math.min(1, (now - this.since) / STABLE_MS)
  }

  reset() {
    this.since = null
  }
}
