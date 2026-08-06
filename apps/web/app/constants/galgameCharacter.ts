// Billing on the 登场角色 roster. `unknown` is deliberately ABSENT: it is a
// meaningful catalog value (a source that publishes no billing, or a character
// reached only through a voice credit), but "未知" is not worth a chip — a miss
// here renders no badge at all, exactly like the staff gender table.
export const KUN_GALGAME_CHARACTER_KIND_MAP: Record<string, string> = {
  main: '主角',
  secondary: '配角',
  appears: '登场'
}

// Only the main cast gets a coloured badge; 配角 / 登场 are ordinary. The roster
// arrives sorted main-first, so the colour is reinforcing the order rather than
// carrying it.
export const KUN_GALGAME_CHARACTER_KIND_COLOR: Record<
  string,
  'primary' | 'default'
> = {
  main: 'primary'
}

// Language codes as they arrive on a character's 简介 and voice credits:
// BCP-47-ish (`ja`, `en`, `zh-Hans`), NOT the `ja-jp` shape the resource /
// original-language maps use. Its own table rather than a reuse that would
// silently miss every row and print the raw code.
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

// The long tail falls back to the raw code: showing `ca` is honest, and
// inventing a name for a code we do not know is not.
export const getGalgameCharacterLangName = (lang: string): string =>
  CHARACTER_LANG_MAP[lang?.toLowerCase()] || lang

// Who wrote a 简介 / owns an external anchor. Attribution, not a link — these
// bios are other databases' prose and the page credits them as such.
const CHARACTER_SOURCE_MAP: Record<string, string> = {
  vndb: 'VNDB',
  bangumi: 'Bangumi',
  getchu: 'Getchu',
  dlsite: 'DLsite'
}

export const getGalgameCharacterSourceName = (source: string): string =>
  CHARACTER_SOURCE_MAP[source?.toLowerCase()] || source

// How much a character's mere presence gives away (VNDB `chars_vns.spoil`).
// Anything above 0 is withheld behind an explicit click.
export const KUN_GALGAME_CHARACTER_SPOILER_MAP: Record<number, string> = {
  1: '轻微剧透',
  2: '严重剧透'
}
