import type {
  KunGalgameResourceTypeOptions,
  KunGalgameResourceLanguageOptions,
  KunGalgameResourcePlatformOptions
} from '~/constants/galgame'
import type { GalgameEngineItem } from './galgame-engine'
import type { GalgameOfficialItem } from './galgame-official'
import type { GalgameTagItem } from './galgame-tag'
import type { GalgameRatingCardOnGalgamePage } from './galgame-rating'

export interface GalgameDetailTag extends GalgameTagItem {
  spoiler_level: number
}

// U2: cover/screenshot row shapes (snake_case wire — matches wiki and
// the kungal DTO; the FE stores them verbatim and round-trips them on
// PUT/PR with presence-replace semantics). `cdn_url` is injected by
// kungal (rewriteBanners walker) — FE never has to hash → URL itself.
export interface GalgameCover {
  image_hash: string
  sort_order: number
  sexual: number
  violence: number
  source: string
  source_key: string
  // VNDB cover type (main/pkgfront/dig/pkgback/…); '' for user uploads.
  kind?: string
  cdn_url?: string
  // Intrinsic display metadata (image_service): pixel dims reserve the aspect
  // ratio (no CLS) + base64 ThumbHash drives the blur-up. Optional — empty for
  // images predating the image_service thumbhash backfill.
  width?: number
  height?: number
  thumbhash?: string
}

export interface GalgameScreenshot extends GalgameCover {
  caption: string
}

export interface GalgameDetail {
  id: number
  vndb_id: string
  user: KunUser
  name: KunLanguage
  banner: string
  introduction: KunLanguage
  content_limit: string
  markdown: KunLanguage
  resource_update_time: Date | string
  // U1 (wiki release_date / release_date_tba): nil = unknown; TBA flag is
  // independent of the date (a TBA entry may still carry a predicted
  // "YYYY-MM-DD"). Server passes through as null when unknown.
  release_date: string | null
  release_date_tba: boolean
  // U2 / K-PR6: covers[sort_order=0] is the canonical banner source.
  // wiki exposes the derived hash; kungal's rewriteBanners walker
  // injects effective_banner_url. (banner_image_hash was retired in
  // wiki PR5; legacy `banner` URL field is still emitted for old data.)
  effective_banner_hash?: string
  effective_banner_url?: string
  // Derived banner's intrinsic metadata (covers[sort_order=0]); also available
  // per-cover. See resolveBannerThumbhash / imageAspectRatio.
  effective_banner_width?: number
  effective_banner_height?: number
  effective_banner_thumbhash?: string
  covers: GalgameCover[]
  screenshots: GalgameScreenshot[]
  view: number
  // false ⇒ a wiki-catalogue galgame the forum hasn't ingested (no local row).
  // The detail page shows a 未收录 notice + hides the (0) view count, keeping the
  // upload/rate/comment CTAs. Absent on older payloads → treated as on-forum.
  is_on_forum?: boolean
  // wiki 草稿状态: 0=已发布, 2=VNDB 草稿(可认领), 3/4=提交者自己的待审/被拒。
  // 未收录提示据此只在可认领的草稿(=2)上展示「认领成为创建者」。
  status?: number
  original_language: string
  age_limit: 'all' | 'r18'
  platform: string[]
  language: string[]
  type: string[]
  contributor: KunUser[]
  like_count: number
  is_liked: boolean
  favorite_count: number
  is_favorited: boolean
  // Moderators have forbidden publishing download resources under this game
  // (copyright / third-party). The resource tab shows a notice + hides publish.
  resource_publish_banned: boolean
  /**
   * DLsite affiliate links for the header's 正版购买 button, assembled server-side
   * from the catalog's `refs.dlsite` work number. Absent when this galgame has no
   * DLsite id or the affiliate is unconfigured — the button then does not render.
   * Never build these in the frontend: the affiliate template is server config.
   */
  dlsite_purchase_url?: string
  dlsite_coupon_url?: string
  alias: string[]
  engine: GalgameEngineItem[]
  official: GalgameOfficialItem[]
  tag: GalgameDetailTag[]
  ratings: GalgameRatingCardOnGalgamePage[]
  created: Date | string
  updated: Date | string
}

export interface GalgamePageRequestData {
  page: string
  limit: string
  type: KunGalgameResourceTypeOptions
  language: KunGalgameResourceLanguageOptions
  platform: KunGalgameResourcePlatformOptions
  sort_field: 'time' | 'views'
  sort_order: KunOrder
}

export interface GalgameCard {
  id: number
  name: KunLanguage
  banner: string
  user: KunUser
  content_limit: string
  view: number
  like_count: number
  // Bayesian-smoothed display rating + vote count. Optional: only the
  // /galgame list endpoint computes them; other card sources omit them
  // (rating_count falsy → the card hides the rating badge).
  rating?: number
  rating_count?: number
  // Entity detail pages (会社/tag/engine/series) list the FULL wiki catalogue.
  // false ⇒ a wiki-catalogue game the forum hasn't ingested (no resources /
  // ratings / views) — the card hides those forum-only fields + shows 未收录.
  // Absent on the /galgame list + other card sources (treated as "on forum").
  is_on_forum?: boolean
  platform: string[]
  language: string[]
  resource_update_time: Date | string
  // U1: optional on card; nil = unknown.
  release_date?: string | null
  release_date_tba?: boolean
  // Release-date precision — only the calendar endpoints emit it (absent
  // elsewhere). Tells the calendar how to read release_date: day = 确切发售日,
  // month = "YYYY-MM-01" 日未定, year = "YYYY-01-01" 月未定, tba/unknown = null.
  // See docs/galgame_wiki/01-galgame.md §release_precision.
  release_precision?: 'day' | 'month' | 'year' | 'tba' | 'unknown'
  // wiki 草稿状态 (calendar only): 2 = 未认领的 VNDB 草稿. The calendar renders
  // status=2 as a "未发布" claim card (→ publish wizard) rather than a detail
  // link (drafts 404 at /galgame/:gid). Absent/0 ⇒ published.
  status?: number
  // U2 / K-PR6: cards carry only the derived banner; URL injected by
  // kungal. banner_image_hash retired in wiki PR5.
  effective_banner_hash?: string
  effective_banner_url?: string
  // Derived banner's intrinsic metadata for no-CLS aspect-ratio + blur-up.
  effective_banner_width?: number
  effective_banner_height?: number
  effective_banner_thumbhash?: string
}

// MineGalgameItem matches the per-row shape of GET /api/galgame/mine.
// Wire-format keeps snake_case (passed through verbatim from wiki).
// See docs/galgame_wiki/07-submission.md §GET /galgame/mine.
//
// Optional name_* / banner_* / vndb_id reflect that wiki's example
// payload only includes name_zh_cn — other languages may simply be
// absent for drafts that don't have a translation yet. Marking these
// required would be a type lie that crashes strict consumers.
export interface MineGalgameItem {
  id: number
  status: number
  vndb_id?: string
  name_en_us?: string
  name_ja_jp?: string
  name_zh_cn?: string
  name_zh_tw?: string
  banner?: string
  content_limit?: string
  // U1: wire is snake_case (verbatim from wiki /galgame/mine).
  release_date?: string | null
  release_date_tba?: boolean
  // U2 / K-PR6: kungal walker injects effective_banner_url on the
  // verbatim wiki wire (banner_image_hash retired in wiki PR5).
  effective_banner_hash?: string
  effective_banner_url?: string
  created: string
  updated: string
  // Only present when status=4 (declined): the latest admin decline
  // reason, lifted from the wiki message payload server-side so the
  // "我的提交" page shows "被拒 + 原因" without a second /messages/mine
  // call. omitempty otherwise. See docs/galgame_wiki/07-submission.md.
  decline_reason?: string
}

export interface MineGalgameList {
  items: MineGalgameItem[]
  total: number
}

// Galgame draft status — see docs/galgame_wiki/07-submission.md §Status 取值.
export const GalgameStatus = {
  Published: 0,
  Banned: 1,
  VndbDraft: 2,
  Pending: 3,
  Declined: 4
} as const

// ──────────────────────────────────────────
// Release calendar (发售月历) — GET /galgame/calendar(/pending|/tba).
// Items are enriched GalgameCard[] (with release_precision + the 未收录 marker).
// See docs/galgame_wiki/01-galgame.md §Galgame 发售月历.
// ──────────────────────────────────────────

// hasPrev/hasNext are data-boundary clamps (no day/month data before minMonth
// or after maxMonth) — the page disables paging at the edges.
export interface GalgameCalendarMeta {
  prev_month: string
  next_month: string
  has_prev: boolean
  has_next: boolean
  min_month: string
  max_month: string
  count: number
}

// One ISO month, already date-sorted by the wiki (exact-day entries first,
// "日未定" month-precision tail last). `today` is JST, for the 今日 marker.
export interface GalgameCalendarMonth {
  month: string
  today: string
  items: GalgameCard[]
  meta: GalgameCalendarMeta
}

// "Year known, month undecided" bucket (release_precision='year') for a year.
export interface GalgameCalendarPending {
  year: string
  items: GalgameCard[]
  count: number
}

// Global "release date to be announced" bucket (release_precision='tba').
export interface GalgameCalendarTBA {
  items: GalgameCard[]
  count: number
}

export interface GalgameCalendarUpcomingMonth {
  month: string
  items: GalgameCard[]
}

// Consolidated "未发售" schedule: every dated entry (day/month precision) with
// release_date >= today, aggregated forward and grouped by month.
export interface GalgameCalendarUpcoming {
  today: string
  months: GalgameCalendarUpcomingMonth[]
  count: number
}
