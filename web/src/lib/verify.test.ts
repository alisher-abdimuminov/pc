import { describe, expect, it } from 'vitest'
import type { Today, TodayStep } from '@/api/student'
import { accuracyLevel, cameraErrorMessage, canVerify, failureGuidance, geoErrorMessage, scaleToFit, secureContextProblem } from './verify'

const step = (over: Partial<TodayStep> = {}): TodayStep => ({
  step: 1, start: '08:00', end: '10:00', state: 'pending', current: true, attempts: 0, max_attempts: 5, ...over,
})
const today = (over: Partial<Today> = {}): Today => ({
  date: '2026-10-12', server_time: '', state: 'active', schedule: null, shift: 1, shift_locked: false,
  steps: [step()], gps_max_accuracy_m: 100, ...over,
})

describe('permission and context messages', () => {
  it('flags insecure contexts', () => {
    expect(secureContextProblem(true)).toBeNull()
    expect(secureContextProblem(false)).toContain('HTTPS')
  })
  it('maps geolocation error codes', () => {
    expect(geoErrorMessage(1)).toContain('ruxsat')
    expect(geoErrorMessage(2)).toContain('GPS')
    expect(geoErrorMessage(3)).toContain('vaqti tugadi')
    expect(geoErrorMessage(99)).toContain('xatolik')
  })
  it('maps camera error names', () => {
    expect(cameraErrorMessage('NotAllowedError')).toContain('ruxsat')
    expect(cameraErrorMessage('NotFoundError')).toContain('topilmadi')
    expect(cameraErrorMessage('NotReadableError')).toContain('band')
    expect(cameraErrorMessage('Whatever')).toContain('xatolik')
  })
})

describe('failureGuidance', () => {
  it('returns a title and a next step for every known failure, in order', () => {
    const g = failureGuidance(['location', 'liveness', 'face'])
    expect(g.map((x) => x.code)).toEqual(['location', 'liveness', 'face'])
    expect(g.every((x) => x.title && x.hint)).toBe(true)
  })
  it('does not crash on unknown codes', () => {
    expect(failureGuidance(['brand_new'])[0]).toEqual({ code: 'brand_new', title: 'brand_new', hint: '' })
    expect(failureGuidance([])).toEqual([])
  })
})

describe('capture helpers', () => {
  it('scales down but never up', () => {
    expect(scaleToFit(2048, 1024, 1024)).toEqual({ width: 1024, height: 512 })
    expect(scaleToFit(600, 800, 1024)).toEqual({ width: 600, height: 800 })
    expect(scaleToFit(1000, 4000, 1000)).toEqual({ width: 250, height: 1000 })
  })
  it('judges GPS accuracy against the server limit', () => {
    expect(accuracyLevel(100, 100)).toBe('good')
    expect(accuracyLevel(100.1, 100)).toBe('poor')
  })
})

describe('canVerify', () => {
  it('is offered for the current pending step', () => {
    expect(canVerify(today())).toEqual({ ok: true, step: 1 })
  })
  it('is not offered when it cannot succeed', () => {
    expect(canVerify(undefined).ok).toBe(false)
    expect(canVerify(today({ state: 'before_start' })).ok).toBe(false)
    expect(canVerify(today({ state: 'finished' })).ok).toBe(false)
    expect(canVerify(today({ shift_locked: true })).ok).toBe(false)
    expect(canVerify(today({ steps: [step({ state: 'done' })] })).ok).toBe(false)
    expect(canVerify(today({ steps: [step({ attempts: 5 })] })).ok).toBe(false)
    expect(canVerify(today({ steps: [step({ current: false })] })).ok).toBe(false)
  })
})
