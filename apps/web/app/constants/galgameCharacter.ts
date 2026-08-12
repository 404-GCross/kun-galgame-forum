export const KUN_GALGAME_CHARACTER_KIND_MAP: Record<string, string> = {
  main: '主角',
  secondary: '配角',
  appears: '登场'
}

export const KUN_GALGAME_CHARACTER_KIND_COLOR: Record<
  string,
  'primary' | 'default'
> = {
  main: 'primary'
}

const CHARACTER_LANG_MAP: Record<string, string> = {
  ja: '日语',
  en: '英语',
  zh: '中文',
  'zh-hans': '简体中文',
  'zh-hant': '繁体中文',
  ko: '韩语',
  ru: '俄语',
  es: '西班牙语',
  fr: '法语',
  de: '德语'
}

export const getGalgameCharacterLangName = (lang: string): string =>
  CHARACTER_LANG_MAP[lang?.toLowerCase()] || lang

const CHARACTER_SOURCE_MAP: Record<string, string> = {
  vndb: 'VNDB',
  bangumi: 'Bangumi',
  getchu: 'Getchu',
  dlsite: 'DLsite'
}

export const getGalgameCharacterSourceName = (source: string): string =>
  CHARACTER_SOURCE_MAP[source?.toLowerCase()] || source

export const KUN_GALGAME_CHARACTER_SPOILER_MAP: Record<number, string> = {
  1: '轻微剧透',
  2: '严重剧透'
}
