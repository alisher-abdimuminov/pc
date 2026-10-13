import { describe, expect, it } from 'vitest'
import { evaluateFace, headPose, LIMITS, StableTimer, STABLE_MS, type Landmark } from './faceGuard'

const rad = (d: number) => (d * Math.PI) / 180
type M3 = number[][]
const mul = (a: M3, b: M3): M3 => a.map((_, i) => [0, 1, 2].map((j) => a[i]![0]! * b[0]![j]! + a[i]![1]! * b[1]![j]! + a[i]![2]! * b[2]![j]!))
const Ry = (d: number): M3 => [[Math.cos(rad(d)), 0, Math.sin(rad(d))], [0, 1, 0], [-Math.sin(rad(d)), 0, Math.cos(rad(d))]]
const Rx = (d: number): M3 => [[1, 0, 0], [0, Math.cos(rad(d)), -Math.sin(rad(d))], [0, Math.sin(rad(d)), Math.cos(rad(d))]]
const Rz = (d: number): M3 => [[Math.cos(rad(d)), -Math.sin(rad(d)), 0], [Math.sin(rad(d)), Math.cos(rad(d)), 0], [0, 0, 1]]

/** 4x4 column-major, like MediaPipe: rotation * scale in the top-left, translation in the last column. */
function matrix(r: M3, scale = 1): number[] {
  const m = new Array<number>(16).fill(0)
  for (let i = 0; i < 3; i++) for (let j = 0; j < 3; j++) m[j * 4 + i] = r[i]![j]! * scale
  m[12] = 3.1
  m[13] = -1.2
  m[14] = -40
  m[15] = 1
  return m
}
const euler = (pitch: number, yaw: number, roll: number): M3 => mul(mul(Ry(yaw), Rx(pitch)), Rz(roll)) // YXZ

/** A face-shaped cloud of points: centre (cx, cy), size w x h. */
const face = (cx: number, cy: number, w: number, h: number): Landmark[] => [
  { x: cx - w / 2, y: cy - h / 2 }, { x: cx + w / 2, y: cy + h / 2 }, { x: cx, y: cy },
]
const straight = matrix(euler(0, 0, 0))

describe('headPose', () => {
  it('is zero for an upright face, whatever the scale or translation', () => {
    for (const scale of [1, 0.4, 3]) {
      const p = headPose(matrix(euler(0, 0, 0), scale))
      expect(Math.abs(p.pitch) + Math.abs(p.yaw) + Math.abs(p.roll)).toBeLessThan(1e-6)
    }
  })

  it('recovers single-axis rotations in degrees', () => {
    expect(headPose(matrix(euler(0, 20, 0))).yaw).toBeCloseTo(20, 5)
    expect(headPose(matrix(euler(15, 0, 0))).pitch).toBeCloseTo(15, 5)
    expect(headPose(matrix(euler(0, 0, 10))).roll).toBeCloseTo(10, 5)
    expect(headPose(matrix(euler(0, -25, 0))).yaw).toBeCloseTo(-25, 5)
  })

  it('recovers combined rotations on a scaled matrix', () => {
    const p = headPose(matrix(euler(-8, 12, -5), 2.5))
    expect(p.pitch).toBeCloseTo(-8, 4)
    expect(p.yaw).toBeCloseTo(12, 4)
    expect(p.roll).toBeCloseTo(-5, 4)
  })

  it('survives a degenerate matrix', () => {
    const p = headPose(new Array(16).fill(0))
    expect(Number.isFinite(p.pitch) && Number.isFinite(p.yaw) && Number.isFinite(p.roll)).toBe(true)
  })
})

describe('evaluateFace', () => {
  it('accepts one centred, well-sized, straight face', () => {
    expect(evaluateFace([face(0.5, 0.5, 0.4, 0.5)], straight)).toEqual({ ok: true, key: 'hold' })
  })

  it('asks for a face when there is none, and for one person when there are several', () => {
    expect(evaluateFace([], straight)).toEqual({ ok: false, key: 'noFace' })
    expect(evaluateFace([face(0.4, 0.5, 0.3, 0.4), face(0.7, 0.5, 0.3, 0.4)], straight)).toEqual({ ok: false, key: 'multiple' })
  })

  it('asks the user to centre, approach or step back', () => {
    expect(evaluateFace([face(0.2, 0.5, 0.3, 0.4)], straight).key).toBe('center')
    expect(evaluateFace([face(0.8, 0.5, 0.3, 0.4)], straight).key).toBe('center')
    expect(evaluateFace([face(0.5, 0.15, 0.3, 0.2)], straight).key).toBe('center')
    expect(evaluateFace([face(0.5, 0.5, 0.15, 0.2)], straight).key).toBe('closer')
    expect(evaluateFace([face(0.5, 0.5, 0.7, 0.5)], straight).key).toBe('farther')
    expect(evaluateFace([face(0.5, 0.5, 0.3, 0.8)], straight).key).toBe('farther')
  })

  it('rejects a head that is turned, tilted or nodding beyond the limit, and accepts small deviations', () => {
    const f = [face(0.5, 0.5, 0.4, 0.5)]
    expect(evaluateFace(f, matrix(euler(0, LIMITS.poseDeg + 3, 0))).key).toBe('straight')
    expect(evaluateFace(f, matrix(euler(-(LIMITS.poseDeg + 3), 0, 0))).key).toBe('straight')
    expect(evaluateFace(f, matrix(euler(0, 0, LIMITS.poseDeg + 3))).key).toBe('straight')
    expect(evaluateFace(f, matrix(euler(3, -4, 2))).ok).toBe(true)
  })

  it('cannot judge the pose without the matrix', () => {
    expect(evaluateFace([face(0.5, 0.5, 0.4, 0.5)], undefined)).toEqual({ ok: false, key: 'noPose' })
  })
})

describe('StableTimer', () => {
  it('counts up while the face is good and reaches 1 after STABLE_MS', () => {
    const t = new StableTimer()
    expect(t.update(true, 1000)).toBe(0)
    expect(t.update(true, 1000 + STABLE_MS / 2)).toBeCloseTo(0.5)
    expect(t.update(true, 1000 + STABLE_MS)).toBe(1)
    expect(t.update(true, 1000 + STABLE_MS * 2)).toBe(1) // never above 1
  })

  it('starts over after any bad frame', () => {
    const t = new StableTimer()
    t.update(true, 0)
    t.update(true, 2500)
    expect(t.update(false, 2600)).toBe(0)
    expect(t.update(true, 2700)).toBe(0)
    expect(t.update(true, 2700 + STABLE_MS - 1)).toBeLessThan(1)
    t.reset()
    expect(t.update(true, 9000)).toBe(0)
  })
})
