// A person's birthday as the registry actually stores it: three independently
// nullable parts, where WHICH parts are present is itself the precision claim.
//
// The registry publishes only what it can stand behind, so all of these are
// real and complete answers: a year alone (1990年), a year and a month, a full
// date, and — for the very common case of a public 誕生日 with no public birth
// year — a month and a day with no year at all (3月4日). Assembling these into
// a Date and formatting that would invent the parts that are missing, which is
// why nothing here ever does.
//
// Anything that cannot be read as a date reads as no date: '' is the answer for
// unknown, and the caller renders no row rather than an empty label.

const inRange = (value: number | null | undefined, lo: number, hi: number) =>
  typeof value === 'number' && Number.isInteger(value) && value >= lo && value <= hi

/**
 * Render a fuzzy birth date in Chinese, from whichever parts are present.
 *
 * - year + month + day → `1990年3月4日`
 * - year + month       → `1990年3月`
 * - year only          → `1990年`
 * - month + day        → `3月4日`   (no year published — a real, common case)
 * - month only         → `3月`
 *
 * A day with no month is dropped: 「4日」 places nothing, and a year plus a day
 * with the month missing is a year. Out-of-range parts (month 13, day 0, year
 * 0 — the shapes a zero-value backend field takes) are treated as absent, so a
 * caller can pass the wire fields through unchecked.
 */
export const formatFuzzyDate = (
  year?: number | null,
  month?: number | null,
  day?: number | null
): string => {
  const y = inRange(year, 1, 9999) ? year! : null
  const m = inRange(month, 1, 12) ? month! : null
  // A day is only meaningful under a month.
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
