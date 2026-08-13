import type {
  EditFieldConfig,
  EditFieldConfigMap,
  EditSelectOption
} from '~/components/editkit/types'
import {
  KUN_GALGAME_OFFICIAL_KIND_OPTIONS,
  KUN_GALGAME_OFFICIAL_KIND_DEVELOPER
} from '~/constants/galgameOfficial'

const K = (name: string) => `catalog.work.${name}`

export interface GalgameEditNames {
  tag?: Map<number, string>
  official?: Map<number, string>
  engine?: Map<number, string>
  series?: Map<number, string>
}

const taxName = (name: unknown): string =>
  typeof name === 'string'
    ? name
    : getPreferredLanguageText(name as never) || ''

interface TaxonomyHit {
  id: number
  name: unknown
}

const searchTaxonomy =
  (path: string) =>
  async (keyword: string): Promise<EditSelectOption[]> => {
    const data = await kunFetch<TaxonomyHit[]>(path, {
      method: 'GET',
      query: { q: keyword }
    })
    return (data ?? []).map((o) => ({ value: o.id, label: taxName(o.name) }))
  }

const browseTaxonomy =
  (path: string) =>
  async (keyword: string): Promise<EditSelectOption[]> => {
    const data = await kunFetch<TaxonomyHit[]>(path, { method: 'GET' })
    const q = keyword.trim().toLowerCase()
    return (data ?? [])
      .map((o) => ({ value: o.id, label: taxName(o.name) }))
      .filter((o) => !q || o.label.toLowerCase().includes(q))
  }

const searchTags = searchTaxonomy('/galgame-tag/search')
const searchOfficials = searchTaxonomy('/galgame-official/search')
const searchEngines = browseTaxonomy('/galgame-engine')
const searchSeries = browseTaxonomy('/galgame-series')

const resolveFrom =
  (map?: Map<number, string>) =>
  (ids: (string | number)[]): EditSelectOption[] => {
    if (!map) {
      return []
    }
    return ids.flatMap((id) => {
      const name = map.get(Number(id))
      return name ? [{ value: Number(id), label: name }] : []
    })
  }

const idFormatItem =
  (map?: Map<number, string>) =>
  (item: unknown): string =>
    map?.get(Number(item)) ?? `#${String(item)}`

const labelFormatItem =
  (map?: Map<number, string>) =>
  (item: unknown): string => {
    const row = item as { label_id?: number; kind?: number }
    const name =
      (row.label_id !== undefined && map?.get(Number(row.label_id))) ||
      `#${row.label_id ?? '?'}`
    const kind = KUN_GALGAME_OFFICIAL_KIND_OPTIONS.find(
      (o) => o.value === row.kind
    )?.label
    return kind ? `${name} · ${kind}` : String(name)
  }

const GROUP_TITLES = '标题'
const GROUP_INTRO = '介绍'
const GROUP_BASIC = '基本信息'
const GROUP_RELATIONS = '关联条目'
const GROUP_EXTRAS = '链接'
const GROUP_IMAGES = '图片'

export const GALGAME_EDIT_GROUP_ORDER = [
  GROUP_TITLES,
  GROUP_INTRO,
  GROUP_BASIC,
  GROUP_RELATIONS,
  GROUP_EXTRAS,
  GROUP_IMAGES
]

export const GALGAME_EDIT_TABBED_GROUPS: string[] = []

export const GALGAME_EDIT_IDENTITY_HINT =
  'VNDB / Bangumi ID 属于条目身份, 不能直接修改。如果发现挂错了, 请在「链接」里补上正确的 VNDB / Bangumi 链接, 并在提交说明里写明原因, 由资料库管理员核对后修正。'

const TITLE_KIND_OPTIONS: EditSelectOption[] = [
  { value: 0, label: '官方名' },
  { value: 1, label: '别名' },
  { value: 2, label: '缩写' }
]

const TITLE_LANG_OPTIONS: EditSelectOption[] = [
  { value: 'ja', label: '日语' },
  { value: 'zh-Hans', label: '简中' },
  { value: 'zh-Hant', label: '繁中' },
  { value: 'en', label: '英语' },
  { value: '', label: '无语言 (别名)' }
]

const INTRO_LANG_OPTIONS: EditSelectOption[] = [
  { value: 'ja', label: '日语' },
  { value: 'zh-Hans', label: '简中' },
  { value: 'zh-Hant', label: '繁中' },
  { value: 'en', label: '英语' }
]

const imageRow = (value: unknown): string => {
  if (value && typeof value === 'object') {
    const row = value as { image_hash?: string }
    if (row.image_hash) {
      return galgameImageSrc({ image_hash: row.image_hash })
    }
  }
  return ''
}

const hasImageHash = (items: unknown[], hash: string): boolean =>
  items.some((item) => (item as { image_hash?: string }).image_hash === hash)

const uploadCoverItem = async (
  file: File,
  current: unknown[]
): Promise<unknown | null> => {
  const res = await uploadGalgameImage(file, 'galgame_banner', file.name)
  if (!res) {
    return null
  }
  if (hasImageHash(current, res.hash)) {
    useMessage('已跳过重复图片', 'warn')
    return null
  }
  return { image_hash: res.hash }
}

const uploadScreenshotItem = async (
  file: File,
  current: unknown[]
): Promise<unknown | null> => {
  const res = await uploadGalgameImage(file, 'galgame_screenshot', file.name)
  if (!res) {
    return null
  }
  if (hasImageHash(current, res.hash)) {
    useMessage('已跳过重复图片', 'warn')
    return null
  }
  return { image_hash: res.hash }
}

export const createGalgameEditConfig = (
  names: GalgameEditNames = {}
): EditFieldConfigMap => ({
  [K('titles')]: {
    label: '标题与别名',
    group: GROUP_TITLES,
    control: 'object-list',
    columns: [
      {
        key: 'lang',
        label: '语言',
        control: 'select',
        options: TITLE_LANG_OPTIONS,
        width: 'w-32'
      },
      { key: 'title', label: '标题', placeholder: '标题 / 别名' },
      {
        key: 'kind',
        label: '类型',
        control: 'select',
        options: TITLE_KIND_OPTIONS,
        width: 'w-28'
      }
    ],
    newRow: () => ({ lang: 'ja', title: '', kind: 0 }),
    formatItem: (item) => {
      const row = item as { lang?: string; title?: string; kind?: number }
      const kind =
        TITLE_KIND_OPTIONS.find((o) => o.value === row.kind)?.label ?? ''
      return `${row.title ?? ''}${row.lang ? ` (${row.lang})` : ''}${kind ? ` · ${kind}` : ''}`
    },
    description: '至少要有一条官方名。别名可以留空语言。'
  },

  [K('display_name')]: {
    label: '条目名',
    group: GROUP_TITLES,
    description: '条目在跨站场合显示的单一名称, 通常与原文官方名一致'
  },

  [K('intros')]: {
    label: '介绍',
    group: GROUP_INTRO,
    control: 'object-list',
    columns: [
      {
        key: 'lang',
        label: '语言',
        control: 'select',
        options: INTRO_LANG_OPTIONS,
        width: 'w-32'
      },
      { key: 'intro', label: '正文', control: 'textarea' }
    ],
    newRow: () => ({ lang: 'zh-Hans', intro: '' }),
    formatItem: (item) => {
      const row = item as { lang?: string; intro?: string }
      return `${row.lang ?? ''}: ${(row.intro ?? '').slice(0, 40)}`
    },
    description: '每种语言一条。图片会被后端剥离。'
  },

  [K('olang')]: {
    label: '原始语言',
    group: GROUP_BASIC,
    options: [
      { value: 'ja', label: '日语' },
      { value: 'zh-Hans', label: '简体中文' },
      { value: 'zh-Hant', label: '繁体中文' },
      { value: 'en', label: '英语' }
    ]
  },
  [K('content_rating')]: {
    label: '年龄分级',
    group: GROUP_BASIC,
    options: [
      { value: 0, label: '全年龄' },
      { value: 1, label: '敏感' },
      { value: 2, label: 'R18' }
    ]
  },
  [K('display_nsfw')]: {
    label: '展示素材为 NSFW',
    group: GROUP_BASIC,
    control: 'switch',
    description: '封面 / 画廊 / 简介是否含成人内容, 与年龄分级是两条独立的轴'
  },

  [K('series_ids')]: {
    label: '所属系列',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchSeries,
    resolveEntities: resolveFrom(names.series),
    formatItem: idFormatItem(names.series),
    description: '搜索并选择所属系列'
  },
  [K('tag_ids')]: {
    label: '标签',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchTags,
    resolveEntities: resolveFrom(names.tag),
    formatItem: idFormatItem(names.tag),
    description: '搜索标签名称添加'
  },
  [K('labels')]: {
    label: '会社',
    group: GROUP_RELATIONS,
    control: 'entity-kind-picker',
    entityIdKey: 'label_id',
    entityKinds: KUN_GALGAME_OFFICIAL_KIND_OPTIONS,
    entityDefaultKind: KUN_GALGAME_OFFICIAL_KIND_DEVELOPER,
    searchEntities: searchOfficials,
    resolveEntities: resolveFrom(names.official),
    formatItem: labelFormatItem(names.official),
    description: '搜索会社名称添加, 并选择它在本作中的身份 (开发商 / 发行商 …)'
  },
  [K('engine_ids')]: {
    label: '引擎',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchEngines,
    resolveEntities: resolveFrom(names.engine),
    formatItem: idFormatItem(names.engine),
    description: '搜索引擎名称添加'
  },

  [K('links')]: {
    label: '相关链接',
    group: GROUP_EXTRAS,
    control: 'string-list',
    placeholder: '粘贴 http(s) 链接后回车添加',
    description: GALGAME_EDIT_IDENTITY_HINT
  },

  [K('covers')]: {
    label: '封面',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    formatItem: (item) => imageRow(item) || JSON.stringify(item),
    uploadImage: uploadCoverItem,
    pinItemFlag: { key: 'portrait_pinned', label: '竖版封面' },
    description:
      '“竖版封面”是卡片和列表渲染的那一张，由这里的置顶决定，与顺序无关；详情页头图由系统在横版图中自动挑选（近正方形的碟面、盒背等永不入选）。拖拽只调整画廊里的展示次序。'
  },
  [K('screenshots')]: {
    label: '画廊',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    formatItem: (item) => imageRow(item) || JSON.stringify(item),
    uploadImage: uploadScreenshotItem,
    description: '拖拽可调整展示顺序'
  }
})

export const GALGAME_EDIT_FIELD_CONFIG: EditFieldConfigMap =
  createGalgameEditConfig()

export const galgameEditLabel = (key: string): string =>
  GALGAME_EDIT_FIELD_CONFIG[key]?.label ?? key.replace('catalog.work.', '')

export const galgameEditFieldConfig = (
  key: string
): EditFieldConfig | undefined => GALGAME_EDIT_FIELD_CONFIG[key]

export const GALGAME_EDIT_LEGACY_ACTION_LABELS: Record<string, string> = {
  created: '创建',
  updated: '更新',
  claimed: '认领',
  approved: '过审',
  declined: '拒绝',
  banned: '封禁',
  unbanned: '解封',
  reverted: '回滚',
  merged: '合并',
  edited_pending: '待审编辑',
  status_changed: '状态变更'
}
