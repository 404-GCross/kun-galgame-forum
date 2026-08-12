import { describe, it, expect } from 'vitest'
import { formatFuzzyDate } from './fuzzyDate'

describe('formatFuzzyDate', () => {
  it('renders each precision the registry can publish', () => {
    expect(formatFuzzyDate(1990, 3, 4)).toBe('1990年3月4日')
    expect(formatFuzzyDate(1990, 3, null)).toBe('1990年3月')
    expect(formatFuzzyDate(1990, null, null)).toBe('1990年')
    expect(formatFuzzyDate(null, 3, 4)).toBe('3月4日')
    expect(formatFuzzyDate(null, 3, null)).toBe('3月')
  })

  it('is empty when nothing is known', () => {
    expect(formatFuzzyDate(null, null, null)).toBe('')
    expect(formatFuzzyDate()).toBe('')
    expect(formatFuzzyDate(undefined, undefined, undefined)).toBe('')
  })

  it('drops a day that no month places', () => {
    expect(formatFuzzyDate(null, null, 4)).toBe('')
    expect(formatFuzzyDate(1990, null, 4)).toBe('1990年')
  })

  it('treats out-of-range parts as absent', () => {
    expect(formatFuzzyDate(0, 0, 0)).toBe('')
    expect(formatFuzzyDate(1990, 13, 4)).toBe('1990年')
    expect(formatFuzzyDate(1990, 3, 32)).toBe('1990年3月')
    expect(formatFuzzyDate(1.5, 3, 4)).toBe('3月4日')
  })
})
