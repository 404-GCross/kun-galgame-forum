import { getPreferredLanguageText } from './getPreferredLanguageText'

export interface WireGalgameName {
  name_en_us?: string
  name_ja_jp?: string
  name_zh_cn?: string
  name_zh_tw?: string
}

export const galgameNameFromWire = (
  g: WireGalgameName,
  fallback = ''
): string => {
  return (
    getPreferredLanguageText({
      'en-us': g.name_en_us ?? '',
      'ja-jp': g.name_ja_jp ?? '',
      'zh-cn': g.name_zh_cn ?? '',
      'zh-tw': g.name_zh_tw ?? ''
    }) || fallback
  )
}
