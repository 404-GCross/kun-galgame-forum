// The 角色 detail page — GET /galgame-character/:id, where `:id` is the CATALOG
// character id the 登场角色 roster already carries.
//
// Bounded by what the catalog's public lane publishes: there is no 性别 / 生日 /
// 三围 here, and there will not be — those live on the staff-side face only.

import type {
  GalgameArtMeta,
  GalgameCard,
  GalgameDetailCharacterVoice
} from './galgame'

export interface GalgameCharacterIntro {
  lang: string
  intro: string
  /** Who wrote it (vndb, bangumi …). Rendered as attribution: this is another
   *  database's prose and the page says so. */
  source: string
  /** An LLM translation, surfaced only for a language no source wrote. Say so
   *  in the UI rather than passing it off as an original. */
  machine: boolean
}

export interface GalgameCharacterTrait {
  id: number
  name: string
  /** The trait's root group ("Hair", "Personality" …); '' when the trait IS a
   *  root. Traits are rendered grouped by it. */
  group: string
  /** 0 = none, 1 = minor, 2 = major. Everything above 0 arrives with the page
   *  and stays hidden until the reader asks — the reveal costs no request. */
  spoiler: number
  /** VNDB's "appears true, is actually false" marker. */
  lie: boolean
}

export interface GalgameCharacterLink {
  source: string
  name: string
  /** Absent for a source with no verified character-page template; the row
   *  then renders as text rather than as a guess that 404s. */
  url?: string
}

/** One game the character appears in — the shared galgame card, plus who
 *  voiced her THERE. Per-work rather than one CV for the character: a recast
 *  between an original and its remake is a real event, not a duplicate. */
export interface GalgameCharacterWork extends GalgameCard {
  /** The registry id, and the only stable key on this list: every work the
   *  forum has not ingested shares gid 0. */
  catalog_id: number
  voices: GalgameDetailCharacterVoice[]
}

export interface GalgameCharacterDetail {
  id: number
  name: string
  name_ja?: string
  name_zh?: string
  latin?: string
  /** Bust portrait, a complete CDN URL, '' for none. Cover-cropped to 256×360
   *  upstream, so it may be rendered in a portrait box. */
  image: string
  /** Full-body 立绘, a complete CDN URL, '' for none. A DIFFERENT ASSET from
   *  `image`, not a larger version of it, and neither falls back to the other.
   *  Render it at its own aspect ratio. */
  figure: string
  /** Each artwork's own shape — absent when image_service could not answer. */
  image_meta?: GalgameArtMeta
  figure_meta?: GalgameArtMeta
  /** The one description the page leads with, already picked server-side. */
  intro: string
  intros: GalgameCharacterIntro[]
  traits: GalgameCharacterTrait[]
  links: GalgameCharacterLink[]
  works: GalgameCharacterWork[]
  /** null on the last page — the only end-of-list signal the catalog gives,
   *  and the reason this page shows a 加载更多 rather than a pager. */
  next_offset: number | null
  /** The ONLY field set when this character id was merged away: the page 301s
   *  to the survivor instead of rendering anything under the dead id. */
  moved_to?: number
}
