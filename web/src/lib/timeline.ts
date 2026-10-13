/** Layout maths for the day timeline on the student home page. */

export const PX_PER_MIN = 3

/** "08:30" -> 510 */
export function toMinutes(hhmm: string): number {
  const [h, m] = hhmm.split(':').map(Number)
  return (h ?? 0) * 60 + (m ?? 0)
}

/** Minutes since midnight in the business timezone (Asia/Tashkent) for an epoch in ms. */
export function businessMinutes(epochMs: number): number {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Asia/Tashkent',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(new Date(epochMs))
  const get = (t: string) => Number(parts.find((p) => p.type === t)?.value ?? 0)
  return get('hour') * 60 + get('minute')
}

export interface Rect {
  top: number
  height: number
}

/** Vertical position/size of each step relative to the start of the first one. */
export function layoutSteps(steps: { start: string; end: string }[], px = PX_PER_MIN): Rect[] {
  if (!steps.length) return []
  const origin = toMinutes(steps[0]!.start)
  return steps.map((s) => ({
    top: (toMinutes(s.start) - origin) * px,
    height: (toMinutes(s.end) - toMinutes(s.start)) * px,
  }))
}

/** Total height of the timeline. */
export function timelineHeight(steps: { start: string; end: string }[], px = PX_PER_MIN): number {
  if (!steps.length) return 0
  return (toMinutes(steps[steps.length - 1]!.end) - toMinutes(steps[0]!.start)) * px
}

/** Hour labels ("08:00") with their offsets, from the first step's start to the last step's end (exclusive). */
export function hourMarks(steps: { start: string; end: string }[], px = PX_PER_MIN): { label: string; top: number }[] {
  if (!steps.length) return []
  const start = toMinutes(steps[0]!.start)
  const end = toMinutes(steps[steps.length - 1]!.end)
  const marks: { label: string; top: number }[] = []
  for (let m = Math.ceil(start / 60) * 60; m < end; m += 60) {
    marks.push({ label: `${String(Math.floor(m / 60)).padStart(2, '0')}:00`, top: (m - start) * px })
  }
  return marks
}

/** Offset of the "now" marker, or null when now is outside the timeline. */
export function markerTop(nowMin: number, steps: { start: string; end: string }[], px = PX_PER_MIN): number | null {
  if (!steps.length) return null
  const start = toMinutes(steps[0]!.start)
  const end = toMinutes(steps[steps.length - 1]!.end)
  if (nowMin < start || nowMin >= end) return null
  return (nowMin - start) * px
}

export const clock = (min: number) => `${String(Math.floor(min / 60)).padStart(2, '0')}:${String(min % 60).padStart(2, '0')}`

export type StepLook = 'done' | 'missed' | 'open' | 'upcoming'

/** How a step card is presented. */
export function stepLook(s: { state: 'done' | 'pending' | 'missed'; current: boolean }): StepLook {
  if (s.state === 'done') return 'done'
  if (s.state === 'missed') return 'missed'
  return s.current ? 'open' : 'upcoming'
}
