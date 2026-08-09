import type { KunGalgameOfficialCategory } from '~/constants/galgameOfficial'
import type { GalgameCard } from './galgame'

export interface GalgameOfficial {
  id: number
  name: string
  link: string
  category: KunGalgameOfficialCategory
  lang: string
}

export interface GalgameOfficialItem {
  id: number
  name: string
  link: string
  category: KunGalgameOfficialCategory
  /** What this label DID on the work it is attached to. Distinct from
   * `category`, which is what kind of organisation it is — one brand can be
   * both developer and publisher of the same title.
   *
   * Optional because this shape is shared with the browse lanes, where a label
   * is not attached to any work and the endpoint sends no roles at all. Present
   * on the galgame detail payload. */
  roles?: string[]
  lang: string
  alias: string[]
  galgame_count: number
  /** The 会社 logo as a ready-made absolute CDN URL (original size — use
   * `withImageVariant(logo, 'mini')` for the thumbnail). `''` when the maker
   * has no logo; optional because the galgame-detail labels[] lane shares this
   * shape and sends none. Always SFW, so no content gate applies. */
  logo?: string
}

// One of a 会社's web presences. `source` is the catalog's own key
// (official_site / twitter / cien) — it NAMES the link, so an X account is
// never rendered as "官方网站".
export interface GalgameOfficialLink {
  source: string
  /** Resolved server-side. Most of these arrive under the catch-all source
   * `web` (wikipedia / youtube / wikidata / …), whose site identity is in the
   * URL rather than in the key, so naming them takes a host table — and that
   * table lives once, in Go, shared with the work and person faces. */
  name: string
  url: string
}

export interface GalgameOfficialDetail {
  id: number
  name: string
  // Original-language name (wiki PR4 sub-change, K-PR6). BE returns ""
  // when wiki hasn't recorded an original yet. The edit modal pre-fills
  // from this so admins can see the existing value instead of starting
  // from empty.
  original: string
  // `link` is the official site alone (empty when the 会社 has none); `links`
  // is every presence the catalog carries, each labelled by its source.
  links: GalgameOfficialLink[]
  link: string
  /** See GalgameOfficialItem.logo. `''` when the maker has no logo — the page
   * then renders the name alone rather than an empty frame. */
  logo: string
  category: KunGalgameOfficialCategory
  lang: string
  description: string
  alias: string[]
  galgame: GalgameCard[]
  galgame_count: number
  /** `galgame_count` split into the works this 会社 is credited with itself and
   * the works that reached the page through one of its imprints or
   * subsidiaries. Shown as two numbers and never added together: a holding
   * company that publishes nothing under its own name reads 0 · 265, and
   * merging them into one 265 is the reassignment `via_official` exists to
   * prevent. They sum to `galgame_count` by construction. */
  own_galgame_count: number
  imprint_galgame_count: number
  // Set — and alone — when this label id was merged away in the catalog: the
  // identity lives on that id now and the page 301s there in one hop. Absent on
  // a live label, so `if (data.moved_to)` is the whole check.
  moved_to?: number
}

/** The catalog's corporate-relation vocabulary — four MUTUALLY INVERSE pairs.
 * The graph face only ever emits the canonical half of each pair
 * (parent / imprint / spawned / succeeded_by); the other four words are what
 * the same edge means read backwards, and are used here only as display keys. */
export type GalgameOfficialRelation =
  | 'parent'
  | 'subsidiary'
  | 'imprint'
  | 'imprint_of'
  | 'spawned'
  | 'origin'
  | 'succeeded_by'
  | 'formerly'

/** One 会社 in the corporate family graph. */
export interface GalgameOfficialRelationNode {
  id: number
  name: string
  /** Ready-made absolute CDN URL like GalgameOfficialDetail.logo; `''` when the
   * maker has no logo, which is the cue to render the name alone. */
  logo: string
  /** The CATALOG-wide count, deliberately NOT the page header's forum-local
   * `galgame_count` — the tree describes the corporate structure, not this
   * site's holdings. */
  work_count: number
}

/** Reads "`to` is the `relation` of `from`" — e.g. {from: Key, to: VisualArt's,
 * relation: 'parent'} means VisualArt's is Key's parent.
 *
 * Note the two ownership words therefore point OPPOSITE ways: a `parent` edge
 * runs child→parent, an `imprint` edge runs owner→brand. */
export interface GalgameOfficialRelationEdge {
  from: number
  to: number
  relation: GalgameOfficialRelation
}

/** The connected component around one 会社 — capped upstream (depth ≤ 4,
 * ≤ 60 nodes), cycle-safe, and ALWAYS containing the requested label itself.
 * So `nodes.length <= 1` means "no recorded relations" and the whole section
 * stays unrendered. */
export interface GalgameOfficialRelationGraph {
  nodes: GalgameOfficialRelationNode[]
  edges: GalgameOfficialRelationEdge[]
}
