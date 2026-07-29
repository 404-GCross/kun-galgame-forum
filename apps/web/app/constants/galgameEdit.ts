// galgame.game presentation config for the schema-driven editor (E3a).
// The eternal field keys are infra's editing-engine contract
// (kun-galgame-infra internal/platform/galgame/editspec/keys.go); this map is
// the HOST side of the editkit boundary — labels, controls, option lists and
// image resolution all live here, never inside components/editkit.

import type {
  EditFieldConfig,
  EditFieldConfigMap,
  EditSelectOption
} from '~/components/editkit/types'
import GalgameEditIntroEditor from '~/components/galgame/edit/IntroEditor.vue'

export const GALGAME_EDIT_ENTITY_TYPE = 'galgame.game'

const K = (name: string) => `galgame.game.${name}`

// ---- taxonomy name pickers (entity-picker control) -------------------------
// The relation fields store ids but the user searches + sees NAMES. Search
// runs against the forum's taxonomy endpoints; the CURRENT ids are resolved
// from the id→name maps the edit page builds off GET /galgame/:gid (which
// already ships tag/official/engine/series resolved). tag/official have a
// Meilisearch /search; engine has none (04-taxonomy.md — filter the full
// list client-side); series proxies the wiki /series/search.

/** id→name map (Map<number, string>) per taxonomy, built from the galgame
 * detail on the edit page and passed into the config factory. */
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

// These four pickers feed galgame.game.{tag,official,engine}_ids / series_id,
// which the editing engine applies to the WIKI galgame row — so they must
// return WIKI ids, and they read the taxonomy picker rather than the public
// browse lanes: those moved to the catalog id space in the A2-3 re-anchoring,
// and feeding a catalog id into a wiki-id edit proposal would silently attach
// the wrong tags rather than fail.
//
// The picker is gated on being signed in, which is the same bar the submission
// form itself sets — so a contributor who can write the field can also fill it.
//
// Blank-query behaviour is the family's, not ours: tag / official return
// nothing until you type (3,037 and 24,334 rows), engine / series return their
// whole curated set (189 and 146).
const taxonomyPicker = (family: string) =>
  searchTaxonomy(`/galgame-taxonomy/${family}/search`)

const searchTags = taxonomyPicker('tag')
const searchOfficials = taxonomyPicker('official')
const searchEngines = taxonomyPicker('engine')
const searchSeries = taxonomyPicker('series')

/** Resolve current ids to {value,label} from a prebuilt map; ids absent from
 * the map are dropped (the picker then shows them as `#id`). */
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

// Groups whose fields collapse behind a per-field sub-tab (SchemaForm
// `tabbedGroups`): the intro group's four languages share one editor + a
// language switch, keeping the page compact.
export const GALGAME_EDIT_TABBED_GROUPS = [GROUP_INTRO]

const imageRow = (value: unknown): string => {
  if (typeof value === 'string') {
    // The banner field is the legacy URL column — already renderable.
    return value
  }
  if (value && typeof value === 'object') {
    const row = value as { cdn_url?: string; image_hash?: string }
    if (row.image_hash) {
      return galgameImageSrc({
        cdn_url: row.cdn_url,
        image_hash: row.image_hash
      })
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
  if (current.some((item) => (item as EditImageItem).image_hash === res.hash)) {
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
  if (current.some((item) => (item as EditImageItem).image_hash === res.hash)) {
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

export const createGalgameEditConfig = (
  names: GalgameEditNames = {}
): EditFieldConfigMap => ({
  [K('name_en_us')]: { label: '英语标题', group: GROUP_TITLES },
  [K('name_ja_jp')]: { label: '日语标题', group: GROUP_TITLES },
  [K('name_zh_cn')]: { label: '简体中文标题', group: GROUP_TITLES },
  [K('name_zh_tw')]: { label: '繁体中文标题', group: GROUP_TITLES },

  // Intros use the image-free markdown editor (component escape hatch). Images
  // are removed at the editor (disableImage) AND stripped by the backend. The
  // four languages share one sub-tab (tabLabel = the short language name).
  [K('intro_en_us')]: {
    label: '英语介绍',
    tabLabel: '英语',
    group: GROUP_INTRO,
    component: GalgameEditIntroEditor
  },
  [K('intro_ja_jp')]: {
    label: '日语介绍',
    tabLabel: '日语',
    group: GROUP_INTRO,
    component: GalgameEditIntroEditor
  },
  [K('intro_zh_cn')]: {
    label: '简体中文介绍',
    tabLabel: '简中',
    group: GROUP_INTRO,
    component: GalgameEditIntroEditor
  },
  [K('intro_zh_tw')]: {
    label: '繁体中文介绍',
    tabLabel: '繁中',
    group: GROUP_INTRO,
    component: GalgameEditIntroEditor
  },

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
    label: '所属系列',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: false,
    searchEntities: searchSeries,
    resolveEntities: resolveFrom(names.series),
    description: '搜索并选择所属系列；留空表示不属于任何系列'
  },
  [K('tag_ids')]: {
    label: '标签',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchTags,
    resolveEntities: resolveFrom(names.tag),
    description: '搜索标签名称添加'
  },
  [K('official_ids')]: {
    label: '会社',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchOfficials,
    resolveEntities: resolveFrom(names.official),
    description: '搜索会社名称添加'
  },
  [K('engine_ids')]: {
    label: '引擎',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchEngines,
    resolveEntities: resolveFrom(names.engine),
    description: '搜索引擎名称添加'
  },

  [K('aliases')]: {
    label: '别名',
    group: GROUP_EXTRAS,
    placeholder: '输入别名后回车添加'
  },
  // control pinned explicitly: the generic list derivation would render a
  // tag input and stringify the {name, link} rows to "[object Object]".
  [K('links')]: {
    label: '相关链接',
    group: GROUP_EXTRAS,
    control: 'link-list'
  },

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
    description: '拖拽排序；第一张为详情页头图，可点“设为封面”置顶'
  },
  [K('screenshots')]: {
    label: '画廊',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    formatItem: (item) => imageRow(item) || JSON.stringify(item),
    uploadImage: uploadScreenshotItem,
    normalizeItems: normalizeImageItems,
    description: '拖拽可调整展示顺序'
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
})

/** Static config with NO taxonomy name maps — the edit page builds its own
 * with createGalgameEditConfig(names) off the galgame detail; other consumers
 * (review workbench, label lookups) use this, where relation pickers still
 * search by name but show current ids as `#id`. */
export const GALGAME_EDIT_FIELD_CONFIG: EditFieldConfigMap =
  createGalgameEditConfig()

/** Field key → Chinese label (falls back to the bare key tail). */
export const galgameEditLabel = (key: string): string =>
  GALGAME_EDIT_FIELD_CONFIG[key]?.label ?? key.replace('galgame.game.', '')

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
