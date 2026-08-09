// catalog.work presentation config for the schema-driven editor.
// The eternal field keys are infra's editing-engine contract
// (kun-galgame-infra internal/platform/catalog/editspec/work.go); this map is
// the HOST side of the editkit boundary — labels, controls, option lists and
// image resolution all live here, never inside components/editkit.
//
// The move off the wiki entity is a RESHAPE, not a rename, and three of the
// differences are visible to the user:
//
//   - Four localized title columns plus an alias list become ONE `titles` list
//     of {lang, title, kind}. Same for the four synopsis columns.
//   - `links` is a list of URLs. The wiki's per-link NAME is gone: the registry
//     classifies a link by where it points, so a hand-typed name was a second,
//     disagreeing answer to a question the URL already settles.
//   - `vndb_id` / `bid` are NOT editable fields and are absent by design.
//     They are identity ANCHORS, not properties: a field edit is reversible and
//     an identity claim is not, so correcting one goes through the report flow
//     (add the right link, say why) and a curator adjudicates it. See the entry
//     hint below.
//
// Also absent, by 03 §6-1 / §3: the release-date trio (the registry expresses
// precision as a nullable date tail on a release row) and `status` (lifecycle
// is claim_state, moved by semantic actions, never by patching a field).

import type {
  EditFieldConfig,
  EditFieldConfigMap,
  EditSelectOption
} from '~/components/editkit/types'
import {
  KUN_GALGAME_OFFICIAL_KIND_OPTIONS,
  KUN_GALGAME_OFFICIAL_KIND_DEVELOPER
} from '~/constants/galgameOfficial'

export const GALGAME_EDIT_ENTITY_TYPE = 'catalog.work'

const K = (name: string) => `catalog.work.${name}`

// ---- taxonomy name pickers (entity-picker control) -------------------------
// The relation fields store ids but the user searches + sees NAMES. Search runs
// against the forum's own catalog-backed endpoints; the CURRENT ids are resolved
// from the id→name maps the edit page builds off GET /galgame/:gid (which
// already ships tag/official/engine/series resolved).

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

// These pickers feed catalog.work.{tag_ids,labels,engine_ids,series_ids}, which
// the editing engine applies to the REGISTRY work — so they must return REGISTRY
// ids. They used to answer /galgame-taxonomy/{family}/search, which served the
// WIKI id space and retired with the wiki in wave-161 P5: since then the pickers
// returned nothing at all, so no tag, 会社, engine or series could be added to
// any entry.
//
// Each now reads the same catalog lane its own browse page reads, which is what
// makes the id spaces agree by construction rather than by comment:
//
//   tag / official → the entity SEARCH lane. Both facets are far too large to
//     enumerate (3,037 tags, 39,652 labels), and both are Meilisearch-backed.
//   engine / series → the BROWSE lane, walked whole (189 and ~600 rows) and
//     filtered in the picker. Neither has a search index of its own, and at this
//     size building one would cost more than it saves.
const searchTaxonomy =
  (path: string) =>
  async (keyword: string): Promise<EditSelectOption[]> => {
    const data = await kunFetch<TaxonomyHit[]>(path, {
      method: 'GET',
      query: { q: keyword }
    })
    return (data ?? []).map((o) => ({ value: o.id, label: taxName(o.name) }))
  }

// A whole-facet walk, filtered here. The list endpoint takes no keyword, so an
// empty query legitimately shows everything — which is the useful behaviour for
// a few-hundred-row curated vocabulary anyway.
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

// Diff/chip display for one relation item: the NAME when the host supplied a
// map covering it, `#id` otherwise — a reviewer should adjudicate on names,
// not registry ids.
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

// No group collapses behind sub-tabs any more: the four language columns each
// tab used to hold are one list field now, so the tab strip would have exactly
// one tab.
export const GALGAME_EDIT_TABBED_GROUPS: string[] = []

/** Where a user reports a wrong VNDB / Bangumi id.
 *
 * The editor deliberately cannot change one. An identity anchor is not a
 * property of the entry — it asserts that this entry and that external record
 * are the same work — and unlike a field edit that assertion is not reversible
 * by whoever made it. So the entry point is the ordinary edit flow with an
 * explanation attached: add the correct link, say why the current one is
 * wrong, and a curator confirms it against the registry's own conflict check. */
export const GALGAME_EDIT_IDENTITY_HINT =
  'VNDB / Bangumi ID 属于条目身份, 不能直接修改。如果发现挂错了, 请在「链接」里补上正确的 VNDB / Bangumi 链接, 并在提交说明里写明原因, 由资料库管理员核对后修正。'

/** Title kinds, mirroring the registry enum. */
const TITLE_KIND_OPTIONS: EditSelectOption[] = [
  { value: 0, label: '官方名' },
  { value: 1, label: '别名' },
  { value: 2, label: '缩写' }
]

/** Languages the registry accepts on a title / synopsis. An alias carries no
 * language at all — it is a string people also call the work by and belongs to
 * none — which is why the empty option is offered rather than hidden. */
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

// ---- image upload hooks (E3b ruling 3) -------------------------------------
// Uploads ride the galgame image proxy (POST /image/galgame → the catalog edit
// image face; bytes land under the image-service `catalog` site since wave
// 169); the editor only carries the returned hash in the patch. Item shapes
// are the engine's STRICT key sets — covers {image_hash, kind?,
// portrait_pinned?, sexual?, violence?}, screenshots {image_hash, caption?,
// sexual?, violence?} — the engine rejects any other key ("unknown key"),
// and ordering is the ARRAY ORDER: sort_order is derived server-side and must
// never be submitted.

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
  // One list, four languages, plus the aliases that used to be their own
  // field. `kind` is what tells them apart, and an alias is the one kind that
  // may carry no language.
  [K('titles')]: {
    label: '标题与别名',
    group: GROUP_TITLES,
    control: 'object-list',
    columns: [
      { key: 'lang', label: '语言', control: 'select', options: TITLE_LANG_OPTIONS, width: 'w-32' },
      { key: 'title', label: '标题', placeholder: '标题 / 别名' },
      { key: 'kind', label: '类型', control: 'select', options: TITLE_KIND_OPTIONS, width: 'w-28' }
    ],
    newRow: () => ({ lang: 'ja', title: '', kind: 0 }),
    formatItem: (item) => {
      const row = item as { lang?: string; title?: string; kind?: number }
      const kind = TITLE_KIND_OPTIONS.find((o) => o.value === row.kind)?.label ?? ''
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
      { key: 'lang', label: '语言', control: 'select', options: INTRO_LANG_OPTIONS, width: 'w-32' },
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
  // Two DIFFERENT axes that used to be spelled alike. content_rating is what
  // the GAME is rated; display_nsfw is whether the material we would render
  // for it (cover, gallery, synopsis) is safe to show. Most r18 games carry
  // editorially safe display material, so collapsing them hides a healthy
  // catalogue.
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
    // NOT entity-picker: this field's wire shape is one object per ATTRIBUTION
    // EDGE ([{label_id, kind}]), because a 会社 can be both the developer and
    // the publisher of the same game and the detail page renders that. An id
    // array cannot say it, which is why the field was unsubmittable while it
    // was configured as a plain picker.
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

  // URLs only. The registry classifies a link by where it points, so a
  // hand-typed name was a second answer to a question the URL already settles.
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

/** Static config with NO taxonomy name maps — pages that can resolve names
 * (the edit page off the galgame detail, the review workbench off detail +
 * by-id lookups) build their own with createGalgameEditConfig(names); this one
 * serves label lookups, where relation items render as `#id`. */
export const GALGAME_EDIT_FIELD_CONFIG: EditFieldConfigMap =
  createGalgameEditConfig()

/** Field key → Chinese label (falls back to the bare key tail). */
export const galgameEditLabel = (key: string): string =>
  GALGAME_EDIT_FIELD_CONFIG[key]?.label ?? key.replace('catalog.work.', '')

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
