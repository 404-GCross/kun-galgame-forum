// @vitest-environment nuxt
import { describe, it, expect } from 'vitest'
import { galgameNameFromWire } from './galgameStatus'

describe('galgameNameFromWire', () => {
  it('returns a localised name when at least one field is set', () => {
    const out = galgameNameFromWire({
      name_zh_cn: '标题',
      name_ja_jp: 'タイトル'
    })
    expect(['标题', 'タイトル']).toContain(out)
  })

  it('returns fallback when all four name fields are missing/empty', () => {
    expect(galgameNameFromWire({}, '(无标题)')).toBe('(无标题)')
    expect(
      galgameNameFromWire(
        {
          name_en_us: '',
          name_ja_jp: '',
          name_zh_cn: '',
          name_zh_tw: ''
        },
        '#1'
      )
    ).toBe('#1')
  })

  it("default fallback is ''", () => {
    expect(galgameNameFromWire({})).toBe('')
  })

  it('snake_case wire keys are normalised to hyphenated locale keys', () => {
    expect(galgameNameFromWire({ name_en_us: 'Title' })).toBeTruthy()
    expect(galgameNameFromWire({ name_ja_jp: 'タイトル' })).toBeTruthy()
    expect(galgameNameFromWire({ name_zh_cn: '简中' })).toBeTruthy()
    expect(galgameNameFromWire({ name_zh_tw: '繁中' })).toBeTruthy()
  })
})
