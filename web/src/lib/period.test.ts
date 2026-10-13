import { describe, expect, it } from 'vitest'
import { addDays, periodRange, shiftAnchor, todayTashkent } from './period'

describe('periodRange', () => {
  it('day is a single date', () => {
    expect(periodRange('day', '2026-09-15')).toEqual({ from: '2026-09-15', to: '2026-09-15' })
  })
  it('week runs Monday to Sunday', () => {
    // 2026-09-16 is a Wednesday
    expect(periodRange('week', '2026-09-16')).toEqual({ from: '2026-09-14', to: '2026-09-20' })
    expect(periodRange('week', '2026-09-14')).toEqual({ from: '2026-09-14', to: '2026-09-20' }) // Monday
    expect(periodRange('week', '2026-09-20')).toEqual({ from: '2026-09-14', to: '2026-09-20' }) // Sunday
  })
  it('month covers the whole calendar month, including leap February and year ends', () => {
    expect(periodRange('month', '2026-09-15')).toEqual({ from: '2026-09-01', to: '2026-09-30' })
    expect(periodRange('month', '2028-02-10')).toEqual({ from: '2028-02-01', to: '2028-02-29' })
    expect(periodRange('month', '2026-12-31')).toEqual({ from: '2026-12-01', to: '2026-12-31' })
  })
  it('custom uses the given range and falls back sensibly', () => {
    expect(periodRange('custom', '2026-09-15', '2026-09-01', '2026-09-10')).toEqual({ from: '2026-09-01', to: '2026-09-10' })
    expect(periodRange('custom', '2026-09-15', '2026-09-01', '')).toEqual({ from: '2026-09-01', to: '2026-09-01' })
    expect(periodRange('custom', '2026-09-15')).toEqual({ from: '2026-09-15', to: '2026-09-15' })
  })
  it('addDays crosses month boundaries', () => {
    expect(addDays('2026-09-30', 1)).toBe('2026-10-01')
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28')
  })
  it('todayTashkent follows the business timezone, not UTC', () => {
    // 20:00 UTC on the 10th is already 01:00 on the 11th in Tashkent (UTC+5)
    expect(todayTashkent(new Date('2026-10-10T20:00:00Z'))).toBe('2026-10-11')
  })
})

describe('shiftAnchor', () => {
  it('moves by a day, a week or a month', () => {
    expect(shiftAnchor('day', '2026-09-15', 1)).toBe('2026-09-16')
    expect(shiftAnchor('day', '2026-09-01', -1)).toBe('2026-08-31')
    expect(shiftAnchor('week', '2026-09-15', 1)).toBe('2026-09-22')
    expect(shiftAnchor('week', '2026-09-15', -1)).toBe('2026-09-08')
    expect(shiftAnchor('month', '2026-09-15', 1)).toBe('2026-10-15')
  })
  it('clamps the day of month and crosses year ends', () => {
    expect(shiftAnchor('month', '2026-03-31', -1)).toBe('2026-02-28')
    expect(shiftAnchor('month', '2028-03-31', -1)).toBe('2028-02-29')
    expect(shiftAnchor('month', '2026-12-10', 1)).toBe('2027-01-10')
    expect(shiftAnchor('month', '2026-01-10', -1)).toBe('2025-12-10')
  })
})
