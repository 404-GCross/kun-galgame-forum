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

// The two chips answer different questions — the ROLE is what this label did on
// THIS work, the CATEGORY is what kind of organisation it is — and both are
// worth showing. But the catalog's two vocabularies overlap, and on exactly
// three pairs the chips end up stating the SAME fact twice: 发行商/发行商,
// 社团/同人社团, 品牌/游戏品牌. Those collapse to one chip.
//
// The pairing is deliberately by KEY, not by rendered text: an upstream census
// found 60,209 rows of developer·publisher + game_brand, which reads as two
// near-synonyms but is genuinely two different facts (who made it vs what kind
// of house it is), and a text-similarity rule would have eaten it.
export const KUN_GALGAME_OFFICIAL_ROLE_CATEGORY_SYNONYM: Record<
  string,
  string
> = {
  circle: 'doujin_circle',
  publisher: 'publisher',
  brand: 'game_brand'
}

// A 会社's web presences used to be named HERE, by a three-entry source→中文
// map. That was right when the catalog rendered exactly three related-link
// kinds and wrong the moment wave 186 widened it: the new rows arrive under the
// catch-all source `web`, which carries no site identity at all, so ブロッコリー's
// wikipedia / wikidata / youtube / gamefaqs links all fell through to the same
// fallback and rendered as four chips reading「web」.
//
// Naming them needs the URL and a host table, so it moved server-side
// (client.LinkDisplayName), where works, 会社 and 人物 share one table instead of
// keeping three that drift. The page reads `link.name`.

// The corporate-relation vocabulary, read as "X 是本会社的 ___". The catalog
// stores four mutually inverse pairs and the graph face ships only the
// canonical half of each, so the four inverse words are reached by reading an
// edge backwards rather than by receiving a row of their own.
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

// The relation graph's own four words, read ALONG the arrow: "A —旗下品牌→ B"
// means B is a brand of A. A separate table from the two above on purpose —
// this one names an EDGE (a direction between two makers), where
// KUN_GALGAME_OFFICIAL_RELATION_MAP names what the other end is to you and
// KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP names what a row is to the row above it.
// Same vocabulary, three different sentences, and collapsing them produced
// arrows labelled 母公司 pointing at the child.
export const KUN_GALGAME_OFFICIAL_GRAPH_EDGE_MAP: Record<string, string> = {
  subsidiary: '子公司',
  imprint: '旗下品牌',
  succession: '更名为',
  spawn: '拆分出'
}

// What a node is TO ITS PARENT in the family tree. Only two words can appear:
// a tree edge is either a parent edge (this node is a subsidiary) or an imprint
// edge (this node is a brand of the node above). A root that owns others is
// labelled 母公司 — it is the top of what the catalog knows, not necessarily of
// the real company.
export const KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP: Record<string, string> = {
  subsidiary: '子公司',
  imprint: '旗下品牌',
  root: '母公司'
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
