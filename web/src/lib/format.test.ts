import { describe, expect, it } from 'vitest'
import { parsePoints, pct } from './format'

describe('format', () => {
  it('parsePoints skips invalid rows', () => {
    expect(parsePoints(['39.1, 66.2', 'oops', '', '40,67'])).toEqual([
      [39.1, 66.2],
      [40, 67],
    ])
  })
  it('pct handles zero totals', () => {
    expect(pct(1, 4)).toBe('25%')
    expect(pct(0, 0)).toBe('—')
  })
})
