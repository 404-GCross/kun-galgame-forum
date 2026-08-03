// P3 retired the PUBLIC series pages because the WIKI's series vocabulary had
// no migration path: 146 entries, 6 of which corresponded to anything in the
// catalog. Half-moving that would have been worse than dropping it.
//
// The public page is back on a different population — the catalog's own
// source-mirrored series facet, the same one works?series_id= filters on and
// every work record's series block points at. That is a complete facet, not the
// half-migration that was refused.
//
// GalgameSeriesSearchItem below still serves the STAFF editor's wiki rows.

import type { GalgameCard } from './galgame'

/** A catalog series entity page: identity + the forum-local, filterable subset
 * of its member works — the same shape the tag / official / engine pages carry,
 * so the four render through one set of components. */
export interface GalgameSeriesDetail {
  id: number
  name: string
  /** ONE intro, already chosen server-side: the catalog keeps every source's
   * text for a series, and stacking them under the title reads as a bug. */
  description: string
  galgame: GalgameCard[]
  galgame_count: number
}

// Wiki /series/search and /series/modal return FULL galgame rows
// (snake_case multi-language `name_<locale>` columns plus a bunch of
// other fields the select widget doesn't need). The widget reads
// `id` for the value and runs the names through `galgameNameFromWire`
// to pick the user-preferred locale. Extra wire fields are tolerated
// but unused.
export interface GalgameSeriesSearchItem {
  id: number
  name_en_us?: string
  name_ja_jp?: string
  name_zh_cn?: string
  name_zh_tw?: string
}

/** One series a game belongs to, as it appears on the game's detail page.
 * Identity only: the member count lives on the series page, and a second count
 * next to the link would eventually disagree with it. */
export interface GalgameDetailSeriesRef {
  id: number
  name: string
}
