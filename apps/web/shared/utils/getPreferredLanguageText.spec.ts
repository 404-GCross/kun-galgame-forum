// @vitest-environment nuxt
import { describe, it, expect } from 'vitest'
import { getPreferredLanguageText } from './getPreferredLanguageText'

const full = {
  'en-us': 'EN',
  'ja-jp': 'JA',
  'zh-cn': 'CN',
  'zh-tw': 'TW'
}

describe('getPreferredLanguageText', () => {
  it('picks the locale itself when present', () => {
    expect(getPreferredLanguageText(full, 'en-us')).toBe('EN')
    expect(getPreferredLanguageText(full, 'ja-jp')).toBe('JA')
    expect(getPreferredLanguageText(full, 'zh-cn')).toBe('CN')
    expect(getPreferredLanguageText(full, 'zh-tw')).toBe('TW')
  })

  it('default locale is zh-cn (when caller omits)', () => {
    expect(getPreferredLanguageText(full)).toBe('CN')
    expect(
      getPreferredLanguageText({
        'en-us': '',
        'ja-jp': '',
        'zh-cn': '',
        'zh-tw': 'TW'
      })
    ).toBe('TW')
  })

  it('falls back through the locale-priority chain when picked field empty', () => {
    expect(
      getPreferredLanguageText(
        { 'en-us': '', 'ja-jp': 'JA', 'zh-cn': 'CN', 'zh-tw': 'TW' },
        'en-us'
      )
    ).toBe('JA')
    expect(
      getPreferredLanguageText(
        { 'en-us': 'EN', 'ja-jp': 'JA', 'zh-cn': '', 'zh-tw': '' },
        'zh-cn'
      )
    ).toBe('JA')
  })

  it("returns '' when every field is empty", () => {
    expect(
      getPreferredLanguageText(
        { 'en-us': '', 'ja-jp': '', 'zh-cn': '', 'zh-tw': '' },
        'zh-cn'
      )
    ).toBe('')
  })

  it('zh-tw prioritises zh-tw → zh-cn (Chinese cluster) over ja-jp / en-us', () => {
    expect(
      getPreferredLanguageText(
        { 'en-us': 'EN', 'ja-jp': 'JA', 'zh-cn': 'CN', 'zh-tw': 'TW' },
        'zh-tw'
      )
    ).toBe('TW')
    expect(
      getPreferredLanguageText(
        { 'en-us': 'EN', 'ja-jp': 'JA', 'zh-cn': 'CN', 'zh-tw': '' },
        'zh-tw'
      )
    ).toBe('CN')
  })

  it('ja-jp priority: ja-jp → en-us → zh-tw → zh-cn', () => {
    expect(
      getPreferredLanguageText(
        { 'en-us': 'EN', 'ja-jp': '', 'zh-cn': 'CN', 'zh-tw': 'TW' },
        'ja-jp'
      )
    ).toBe('EN')
  })
})
