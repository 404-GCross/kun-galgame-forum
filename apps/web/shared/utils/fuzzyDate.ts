const inRange = (value: number | null | undefined, lo: number, hi: number) =>
  typeof value === 'number' &&
  Number.isInteger(value) &&
  value >= lo &&
  value <= hi

export const formatFuzzyDate = (
  year?: number | null,
  month?: number | null,
  day?: number | null
): string => {
  const y = inRange(year, 1, 9999) ? year! : null
  const m = inRange(month, 1, 12) ? month! : null
  const d = m !== null && inRange(day, 1, 31) ? day! : null

  let out = y !== null ? `${y}年` : ''
  if (m !== null) {
    out += `${m}月`
    if (d !== null) {
      out += `${d}日`
    }
  }
  return out
}
