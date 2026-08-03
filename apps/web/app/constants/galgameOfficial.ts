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

// The EDIT face's label-kind vocabulary. Numeric, unlike the READ face's
// strings (catalog_work_label.kind is an int16 and the edit field carries it
// verbatim) — so the two axes of the same fact are spelled differently on the
// two faces, and this is the only place that has to know it.
export const KUN_GALGAME_OFFICIAL_KIND_CIRCLE = 0
export const KUN_GALGAME_OFFICIAL_KIND_PUBLISHER = 1
export const KUN_GALGAME_OFFICIAL_KIND_DEVELOPER = 2
export const KUN_GALGAME_OFFICIAL_KIND_BRAND = 3

// Offer order is by how often an editor reaches for it, not by the enum's
// numbering: most entries name who made the game and who put it out.
export const KUN_GALGAME_OFFICIAL_KIND_OPTIONS = [
  { value: KUN_GALGAME_OFFICIAL_KIND_DEVELOPER, label: '开发商' },
  { value: KUN_GALGAME_OFFICIAL_KIND_PUBLISHER, label: '发行商' },
  { value: KUN_GALGAME_OFFICIAL_KIND_CIRCLE, label: '社团' },
  { value: KUN_GALGAME_OFFICIAL_KIND_BRAND, label: '品牌' }
]
