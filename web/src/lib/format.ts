export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('uz-UZ', {
    dateStyle: 'short',
    timeStyle: 'medium',
    timeZone: 'Asia/Tashkent',
  }).format(new Date(iso))
}

export function pct(part: number, total: number): string {
  return total ? `${Math.round((part / total) * 100)}%` : '—'
}

/** Parses "39.1,66.2" rows into Leaflet-friendly tuples, skipping invalid rows. */
export function parsePoints(rows: string[]): [number, number][] {
  const out: [number, number][] = []
  for (const r of rows) {
    const [a, b] = r.split(',').map((s) => Number(s.trim()))
    if (Number.isFinite(a) && Number.isFinite(b) && r.includes(',')) out.push([a!, b!])
  }
  return out
}

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('uz-UZ', { dateStyle: 'medium', timeZone: 'Asia/Tashkent' }).format(new Date(iso))
}

export function formatTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('uz-UZ', { timeStyle: 'short', timeZone: 'Asia/Tashkent', hour12: false }).format(new Date(iso))
}
