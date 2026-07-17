// galgame.game presentation config for the schema-driven editor (E3a).
// The eternal field keys are infra's editing-engine contract
// (kun-galgame-infra internal/platform/galgame/editspec/keys.go); this map is
// the HOST side of the editkit boundary — labels, controls, option lists and
// image resolution all live here, never inside components/editkit.

import type {
  EditFieldConfig,
  EditFieldConfigMap
} from '~/components/editkit/types'

export const GALGAME_EDIT_ENTITY_TYPE = 'galgame.game'

const K = (name: string) => `galgame.game.${name}`

const GROUP_TITLES = '标题'
const GROUP_INTRO = '介绍'
const GROUP_BASIC = '基本信息'
const GROUP_RELATIONS = '关联条目'
const GROUP_EXTRAS = '别名与链接'
const GROUP_IMAGES = '图片'
const GROUP_IDENTITY = '标识与管理'

export const GALGAME_EDIT_GROUP_ORDER = [
  GROUP_TITLES,
  GROUP_INTRO,
  GROUP_BASIC,
  GROUP_RELATIONS,
  GROUP_EXTRAS,
  GROUP_IMAGES,
  GROUP_IDENTITY
]

const imageRow = (value: unknown): string => {
  if (typeof value === 'string') {
    // The banner field is the legacy URL column — already renderable.
    return value
  }
  if (value && typeof value === 'object') {
    const row = value as { cdn_url?: string; image_hash?: string }
    if (row.image_hash) {
      return galgameImageSrc({ cdn_url: row.cdn_url, image_hash: row.image_hash })
    }
  }
  return ''
}

// ---- image upload hooks (E3b ruling 3) -------------------------------------
// Uploads ride the existing galgame image proxy (POST /image/galgame →
// wiki → image_service under site=galgame_wiki, the byte-owner iron rule);
// the editor only carries the returned hash/URL in the patch. Item shapes
// mirror the engine's SnapshotCover / SnapshotScreenshot STRICT key sets —
// never add render-only keys (e.g. cdn_url), the engine rejects unknown keys.

interface EditImageItem {
  image_hash: string
  sort_order: number
  [key: string]: unknown
}

/** Stamp sort_order = index — item 0 is the pinned cover (the wiki's
 * "at most one sort_order=0" invariant holds by construction). */
const normalizeImageItems = (items: unknown[]): unknown[] =>
  items.map((item, index) => ({
    ...(item as EditImageItem),
    sort_order: index
  }))

const uploadBanner = async (file: File): Promise<unknown | null> => {
  const res = await uploadGalgameImage(file, 'galgame_banner', file.name)
  return res ? res.url : null
}

const uploadCoverItem = async (
  file: File,
  current: unknown[]
): Promise<unknown | null> => {
  const res = await uploadGalgameImage(file, 'galgame_banner', file.name)
  if (!res) {
    return null
  }
  if (
    current.some((item) => (item as EditImageItem).image_hash === res.hash)
  ) {
    useMessage('已跳过重复图片', 'warn')
    return null
  }
  return {
    image_hash: res.hash,
    sort_order: current.length,
    sexual: 0,
    violence: 0,
    source: '',
    source_key: '',
    kind: ''
  }
}

const uploadScreenshotItem = async (
  file: File,
  current: unknown[]
): Promise<unknown | null> => {
  const res = await uploadGalgameImage(file, 'galgame_screenshot', file.name)
  if (!res) {
    return null
  }
  if (
    current.some((item) => (item as EditImageItem).image_hash === res.hash)
  ) {
    useMessage('已跳过重复图片', 'warn')
    return null
  }
  return {
    image_hash: res.hash,
    sort_order: current.length,
    caption: '',
    sexual: 0,
    violence: 0,
    source: '',
    source_key: ''
  }
}

export const GALGAME_EDIT_FIELD_CONFIG: EditFieldConfigMap = {
  [K('name_en_us')]: { label: '英语标题', group: GROUP_TITLES },
  [K('name_ja_jp')]: { label: '日语标题', group: GROUP_TITLES },
  [K('name_zh_cn')]: { label: '简体中文标题', group: GROUP_TITLES },
  [K('name_zh_tw')]: { label: '繁体中文标题', group: GROUP_TITLES },

  [K('intro_en_us')]: { label: '英语介绍', group: GROUP_INTRO },
  [K('intro_ja_jp')]: { label: '日语介绍', group: GROUP_INTRO },
  [K('intro_zh_cn')]: { label: '简体中文介绍', group: GROUP_INTRO },
  [K('intro_zh_tw')]: { label: '繁体中文介绍', group: GROUP_INTRO },

  [K('release_date')]: {
    label: '发售日期',
    group: GROUP_BASIC,
    control: 'date',
    nullable: true
  },
  [K('release_date_tba')]: {
    label: '发售日待定 (TBA)',
    group: GROUP_BASIC,
    control: 'switch'
  },
  [K('release_precision')]: {
    label: '发售日精度',
    group: GROUP_BASIC,
    options: [
      { value: '', label: '未指定' },
      { value: 'day', label: '精确到日' },
      { value: 'month', label: '精确到月' },
      { value: 'year', label: '精确到年' },
      { value: 'tba', label: '待定' },
      { value: 'unknown', label: '未知' }
    ]
  },
  [K('original_language')]: {
    label: '原始语言',
    group: GROUP_BASIC,
    options: [
      { value: 'ja-jp', label: '日语' },
      { value: 'zh-cn', label: '简体中文' },
      { value: 'zh-tw', label: '繁体中文' },
      { value: 'en-us', label: '英语' }
    ]
  },
  [K('age_limit')]: {
    label: '年龄分级',
    group: GROUP_BASIC,
    options: [
      { value: 'all', label: '全年龄' },
      { value: 'r18', label: 'R18' }
    ]
  },
  [K('content_limit')]: {
    label: '内容限制',
    group: GROUP_BASIC,
    options: [
      { value: 'sfw', label: 'SFW' },
      { value: 'nsfw', label: 'NSFW' }
    ]
  },

  [K('series_id')]: {
    label: '所属系列 ID',
    group: GROUP_RELATIONS,
    nullable: true,
    description: '留空表示不属于任何系列'
  },
  [K('tag_ids')]: {
    label: '标签 ID',
    group: GROUP_RELATIONS,
    control: 'number-list',
    description: '输入标签数字 ID 后回车添加'
  },
  [K('official_ids')]: {
    label: '会社 ID',
    group: GROUP_RELATIONS,
    control: 'number-list',
    description: '输入会社数字 ID 后回车添加'
  },
  [K('engine_ids')]: {
    label: '引擎 ID',
    group: GROUP_RELATIONS,
    control: 'number-list',
    description: '输入引擎数字 ID 后回车添加'
  },

  [K('aliases')]: {
    label: '别名',
    group: GROUP_EXTRAS,
    placeholder: '输入别名后回车添加'
  },
  // control pinned explicitly: the generic list derivation would render a
  // tag input and stringify the {name, link} rows to "[object Object]".
  [K('links')]: { label: '相关链接', group: GROUP_EXTRAS, control: 'link-list' },

  [K('banner')]: {
    label: '横幅图',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    uploadImage: uploadBanner
  },
  [K('covers')]: {
    label: '封面',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    formatItem: (item) => imageRow(item) || JSON.stringify(item),
    uploadImage: uploadCoverItem,
    normalizeItems: normalizeImageItems,
    pinFirstLabel: '封面',
    description: '第一张为详情页头图（钉住的封面），其余为备选'
  },
  [K('screenshots')]: {
    label: '画廊',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    formatItem: (item) => imageRow(item) || JSON.stringify(item),
    uploadImage: uploadScreenshotItem,
    normalizeItems: normalizeImageItems
  },

  [K('vndb_id')]: {
    label: 'VNDB ID',
    group: GROUP_IDENTITY,
    placeholder: '形如 v12345'
  },
  [K('bid')]: { label: 'Bangumi ID', group: GROUP_IDENTITY, nullable: true },
  [K('status')]: {
    label: '状态',
    group: GROUP_IDENTITY,
    options: [
      { value: 0, label: '已发布' },
      { value: 1, label: '已封禁' },
      { value: 2, label: 'VNDB 草稿' },
      { value: 3, label: '待审核' },
      { value: 4, label: '已拒绝' }
    ]
  }
}

/** Field key → Chinese label (falls back to the bare key tail). */
export const galgameEditLabel = (key: string): string =>
  GALGAME_EDIT_FIELD_CONFIG[key]?.label ??
  key.replace('galgame.game.', '')

export const galgameEditFieldConfig = (
  key: string
): EditFieldConfig | undefined => GALGAME_EDIT_FIELD_CONFIG[key]

/** Legacy (pre-engine) action words → labels for the migrated-history badge. */
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
