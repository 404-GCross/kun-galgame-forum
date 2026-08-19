import { describe, expect, it } from 'vitest'
import { getGalgameCharacterIntroCredit } from './galgameCharacter'

describe('getGalgameCharacterIntroCredit', () => {
  it('names a translated blurb as machine-written', () => {
    expect(
      getGalgameCharacterIntroCredit({ source: 'bangumi', machine: true })
    ).toBe('简介来自 Bangumi · 由机器翻译生成')
  })

  it('credits a source blurb without claiming a machine wrote it', () => {
    expect(
      getGalgameCharacterIntroCredit({ source: 'vndb', machine: false })
    ).toBe('简介来自 VNDB')
  })

  it('does not call a derived row a translation', () => {
    // A `derived` row is machine-provenance but verbatim: it is lifted out of
    // the game's own blurb, not translated. Rendered through the generic path
    // it read "简介来自 derived · 由机器翻译生成" — a raw source key plus a claim
    // about the text that was false.
    const credit = getGalgameCharacterIntroCredit({
      source: 'derived',
      machine: true
    })
    expect(credit).not.toContain('derived')
    expect(credit).not.toContain('翻译')
    expect(credit).toBe('摘自本作简介中该角色的段落')
  })

  it('says nothing when there is no blurb to credit', () => {
    expect(getGalgameCharacterIntroCredit(null)).toBe('')
    expect(getGalgameCharacterIntroCredit({})).toBe('')
  })
})
