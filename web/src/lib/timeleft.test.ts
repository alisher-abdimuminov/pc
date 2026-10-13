import { describe, expect, it } from 'vitest'
import { clockOffset, formatTimeLeft } from './timeleft'

const MIN = 60_000
const HOUR = 60 * MIN
const DAY = 24 * HOUR

describe('formatTimeLeft', () => {
  it('reports a passed deadline for zero or negative time', () => {
    expect(formatTimeLeft(0)).toBe('Muddat tugagan')
    expect(formatTimeLeft(-5 * MIN)).toBe('Muddat tugagan')
  })
  it('formats days, hours, minutes and seconds', () => {
    expect(formatTimeLeft(2 * DAY + 3 * HOUR + 10 * MIN)).toBe('2 kun 3 soat')
    expect(formatTimeLeft(2 * DAY)).toBe('2 kun')
    expect(formatTimeLeft(5 * HOUR + 12 * MIN)).toBe('5 soat 12 daqiqa')
    expect(formatTimeLeft(3 * HOUR)).toBe('3 soat')
    expect(formatTimeLeft(42 * MIN + 30_000)).toBe('42 daqiqa')
    expect(formatTimeLeft(45_000)).toBe('45 soniya')
    expect(formatTimeLeft(1)).toBe('1 soniya')
  })
})

describe('clockOffset', () => {
  it('is server minus client time', () => {
    expect(clockOffset('2026-10-12T10:00:10Z', Date.parse('2026-10-12T10:00:00Z'))).toBe(10_000)
    expect(clockOffset('2026-10-12T10:00:00Z', Date.parse('2026-10-12T10:00:30Z'))).toBe(-30_000)
  })
})
