export const KUN_GALGAME_OFFICIAL_TYPE = [
  'company',
  'individual',
  'amateur'
] as const

export type KunGalgameOfficialCategory =
  (typeof KUN_GALGAME_OFFICIAL_TYPE)[number]

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

export const KUN_GALGAME_OFFICIAL_ROLE_MAP: Record<string, string> = {
  developer: '开发商',
  publisher: '发行商',
  circle: '社团',
  brand: '品牌'
}

export const KUN_GALGAME_OFFICIAL_ROLE_CATEGORY_SYNONYM: Record<
  string,
  string
> = {
  circle: 'doujin_circle',
  publisher: 'publisher',
  brand: 'game_brand'
}

export const KUN_GALGAME_OFFICIAL_RELATION_MAP: Record<string, string> = {
  parent: '母公司',
  subsidiary: '子公司',
  imprint: '旗下品牌',
  imprint_of: '所属公司',
  spawned: '拆分出的公司',
  origin: '前身公司',
  succeeded_by: '继任公司',
  formerly: '旧名'
}

export const KUN_GALGAME_OFFICIAL_GRAPH_EDGE_MAP: Record<string, string> = {
  subsidiary: '子公司',
  imprint: '旗下品牌',
  succession: '更名为',
  spawn: '拆分出'
}

export const KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP: Record<string, string> = {
  subsidiary: '子公司',
  imprint: '旗下品牌',
  root: '母公司'
}

export const KUN_GALGAME_OFFICIAL_LANGUAGE_MAP: Record<string, string> = {
  ja: '日语',
  zh: '中文',
  en: '英语',
  id: '印度尼西亚语',
  ko: '韩语',
  ru: '俄语',
  es: '西班牙语'
}

export const KUN_GALGAME_OFFICIAL_KIND_CIRCLE = 0
export const KUN_GALGAME_OFFICIAL_KIND_PUBLISHER = 1
export const KUN_GALGAME_OFFICIAL_KIND_DEVELOPER = 2
export const KUN_GALGAME_OFFICIAL_KIND_BRAND = 3

export const KUN_GALGAME_OFFICIAL_KIND_OPTIONS = [
  { value: KUN_GALGAME_OFFICIAL_KIND_DEVELOPER, label: '开发商' },
  { value: KUN_GALGAME_OFFICIAL_KIND_PUBLISHER, label: '发行商' },
  { value: KUN_GALGAME_OFFICIAL_KIND_CIRCLE, label: '社团' },
  { value: KUN_GALGAME_OFFICIAL_KIND_BRAND, label: '品牌' }
]
