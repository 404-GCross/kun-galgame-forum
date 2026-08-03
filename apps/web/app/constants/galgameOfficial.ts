export const KUN_GALGAME_OFFICIAL_TYPE = [
  'company',
  'individual',
  'amateur'
] as const

export type KunGalgameOfficialCategory =
  (typeof KUN_GALGAME_OFFICIAL_TYPE)[number]

// The public browse pages render the CATALOG label kinds (a richer, better
// defined vocabulary than the wiki's free-text one); the first three keys are
// the wiki values the staff edit form still writes.
export const KUN_GALGAME_OFFICIAL_CATEGORY_MAP: Record<string, string> = {
  company: '公司',
  individual: '个人',
  amateur: '业余',
  game_brand: '游戏品牌',
  bunko: '文库',
  publisher: '发行商',
  anime_studio: '动画工作室',
  doujin_circle: '同人社团',
  group: '团体',
  other: '其它'
}

// The label's ROLE on a given work — a different axis from the category map
// above (what kind of organisation it is). A2-R2 dropped the role from the
// detail chip precisely because this table did not exist and the raw English
// "developer" leaked through; it exists now, so the role renders again.
export const KUN_GALGAME_OFFICIAL_ROLE_MAP: Record<string, string> = {
  developer: '开发商',
  publisher: '发行商',
  circle: '社团',
  brand: '品牌'
}

// /migrate/getAllOfficialLanguage.js
export const KUN_GALGAME_OFFICIAL_LANGUAGE_MAP: Record<string, string> = {
  ja: '日语',
  zh: '中文',
  en: '英语',
  id: '印度尼西亚语',
  ko: '韩语',
  ru: '俄语',
  es: '西班牙语'
}
