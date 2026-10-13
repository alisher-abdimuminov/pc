import { describe, expect, it } from 'vitest'
import { businessMinutes, clock, hourMarks, layoutSteps, markerTop, stepLook, timelineHeight, toMinutes } from './timeline'

const shift1 = [
  { start: '08:00', end: '10:00' },
  { start: '10:00', end: '12:00' },
  { start: '12:00', end: '13:00' },
]
const shift2 = [
  { start: '13:00', end: '14:00' },
  { start: '14:00', end: '16:00' },
  { start: '16:00', end: '18:00' },
]

describe('timeline layout', () => {
  it('parses clock strings', () => {
    expect(toMinutes('08:30')).toBe(510)
    expect(clock(510)).toBe('08:30')
    expect(clock(0)).toBe('00:00')
  })

  it('positions steps proportionally to their duration', () => {
    expect(layoutSteps(shift1, 2)).toEqual([
      { top: 0, height: 240 },
      { top: 240, height: 240 },
      { top: 480, height: 120 },
    ])
    // shift 2 is measured from its own start, so the layout is the same shape shifted to 13:00
    expect(layoutSteps(shift2, 2)[1]).toEqual({ top: 120, height: 240 })
    expect(timelineHeight(shift1, 2)).toBe(600)
    expect(layoutSteps([])).toEqual([])
    expect(timelineHeight([])).toBe(0)
  })

  it('marks every hour inside the shift but not the closing one', () => {
    expect(hourMarks(shift1, 2).map((m) => m.label)).toEqual(['08:00', '09:00', '10:00', '11:00', '12:00'])
    expect(hourMarks(shift1, 2)[1]).toEqual({ label: '09:00', top: 120 })
    expect(hourMarks(shift2, 2).map((m) => m.label)).toEqual(['13:00', '14:00', '15:00', '16:00', '17:00'])
  })

  it('places the now marker, and hides it outside the day', () => {
    expect(markerTop(toMinutes('10:00'), shift1, 2)).toBe(240)
    expect(markerTop(toMinutes('08:00'), shift1, 2)).toBe(0)
    expect(markerTop(toMinutes('12:59'), shift1, 2)).toBe(598)
    expect(markerTop(toMinutes('07:59'), shift1, 2)).toBeNull()
    expect(markerTop(toMinutes('13:00'), shift1, 2)).toBeNull() // end is exclusive
    expect(markerTop(600, [], 2)).toBeNull()
  })

  it('reads the business time zone, not the device one', () => {
    // 05:00 UTC is 10:00 in Tashkent (UTC+5)
    expect(businessMinutes(Date.parse('2026-10-12T05:00:00Z'))).toBe(600)
    // 20:30 UTC is 01:30 the next day in Tashkent
    expect(businessMinutes(Date.parse('2026-10-12T20:30:00Z'))).toBe(90)
  })
})

describe('stepLook', () => {
  it('maps server state and the current flag to a presentation', () => {
    expect(stepLook({ state: 'done', current: false })).toBe('done')
    expect(stepLook({ state: 'done', current: true })).toBe('done')
    expect(stepLook({ state: 'missed', current: false })).toBe('missed')
    expect(stepLook({ state: 'pending', current: true })).toBe('open')
    expect(stepLook({ state: 'pending', current: false })).toBe('upcoming')
  })
})
