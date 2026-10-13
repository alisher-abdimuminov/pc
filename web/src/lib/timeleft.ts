/** Human readable remaining time in Uzbek; non-positive input means the deadline has passed. */
export function formatTimeLeft(ms: number, passed = 'Muddat tugagan'): string {
  if (ms <= 0) return passed
  const totalMin = Math.floor(ms / 60000)
  const days = Math.floor(totalMin / 1440)
  const hours = Math.floor((totalMin % 1440) / 60)
  const mins = totalMin % 60
  if (days > 0) return hours > 0 ? `${days} kun ${hours} soat` : `${days} kun`
  if (hours > 0) return mins > 0 ? `${hours} soat ${mins} daqiqa` : `${hours} soat`
  if (mins > 0) return `${mins} daqiqa`
  return `${Math.max(1, Math.ceil(ms / 1000))} soniya`
}

/** Offset (ms) to add to Date.now() so it matches the server clock. */
export function clockOffset(serverTime: string, clientNow = Date.now()): number {
  return new Date(serverTime).getTime() - clientNow
}
