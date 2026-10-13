export type PeriodMode = 'day' | 'week' | 'month' | 'custom'

const pad = (n: number) => String(n).padStart(2, '0')
const iso = (d: Date) => `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`
const parse = (s: string) => {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(Date.UTC(y!, m! - 1, d!))
}

export function addDays(date: string, n: number): string {
  const d = parse(date)
  d.setUTCDate(d.getUTCDate() + n)
  return iso(d)
}

/** Today's date in the business timezone (Asia/Tashkent), as YYYY-MM-DD. */
export function todayTashkent(now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Tashkent' }).format(now)
}

/** Resolves a period selection into an inclusive [from, to] range. Weeks run Monday-Sunday. */
export function periodRange(mode: PeriodMode, anchor: string, customFrom = '', customTo = ''): { from: string; to: string } {
  switch (mode) {
    case 'day':
      return { from: anchor, to: anchor }
    case 'week': {
      const dow = (parse(anchor).getUTCDay() + 6) % 7 // Monday = 0
      const from = addDays(anchor, -dow)
      return { from, to: addDays(from, 6) }
    }
    case 'month': {
      const d = parse(anchor)
      const from = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), 1))
      const to = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + 1, 0))
      return { from: iso(from), to: iso(to) }
    }
    default:
      return { from: customFrom || anchor, to: customTo || customFrom || anchor }
  }
}

/** Moves the anchor one period back (-1) or forward (+1): a day, a week or a calendar month. */
export function shiftAnchor(mode: PeriodMode, anchor: string, dir: -1 | 1): string {
  switch (mode) {
    case 'day':
      return addDays(anchor, dir)
    case 'month': {
      const d = parse(anchor)
      const first = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + dir, 1))
      const last = new Date(Date.UTC(first.getUTCFullYear(), first.getUTCMonth() + 1, 0)).getUTCDate()
      first.setUTCDate(Math.min(d.getUTCDate(), last)) // Mar 31 -> Feb 28, not Mar 3
      return iso(first)
    }
    default: // week (and custom ranges, which have their own pickers)
      return addDays(anchor, 7 * dir)
  }
}
