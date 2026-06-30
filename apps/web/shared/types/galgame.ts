import type {
  KunGalgameResourceTypeOptions,
  KunGalgameResourceLanguageOptions,
  KunGalgameResourcePlatformOptions
} from '~/constants/galgame'
import type { GalgameSeries } from './galgame-series'
import type { GalgameEngineItem } from './galgame-engine'
import type { GalgameOfficialItem } from './galgame-official'
import type { GalgameTagItem } from './galgame-tag'
import type { GalgameRatingCardOnGalgamePage } from './galgame-rating'

export interface GalgameDetailTag extends GalgameTagItem {
  spoilerLevel: number
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
  vndbId: string
  user: KunUser
  name: KunLanguage
  banner: string
  introduction: KunLanguage
  contentLimit: string
  markdown: KunLanguage
  resourceUpdateTime: Date | string
  // U1 (wiki release_date / release_date_tba): nil = unknown; TBA flag is
  // independent of the date (a TBA entry may still carry a predicted
  // "YYYY-MM-DD"). Server passes through as null when unknown.
  releaseDate: string | null
  releaseDateTBA: boolean
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
  isOnForum?: boolean
  originalLanguage: string
  ageLimit: 'all' | 'r18'
  platform: string[]
  language: string[]
  type: string[]
  contributor: KunUser[]
  likeCount: number
  isLiked: boolean
  favoriteCount: number
  isFavorited: boolean
  alias: string[]
  series: GalgameSeries | null
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
  sortField: 'time' | 'views'
  sortOrder: KunOrder
}

export interface GalgameCard {
  id: number
  name: KunLanguage
  banner: string
  user: KunUser
  contentLimit: string
  view: number
  likeCount: number
  // Bayesian-smoothed display rating + vote count. Optional: only the
  // /galgame list endpoint computes them; other card sources omit them
  // (ratingCount falsy → the card hides the rating badge).
  rating?: number
  ratingCount?: number
  // Entity detail pages (会社/tag/engine/series) list the FULL wiki catalogue.
  // false ⇒ a wiki-catalogue game the forum hasn't ingested (no resources /
  // ratings / views) — the card hides those forum-only fields + shows 未收录.
  // Absent on the /galgame list + other card sources (treated as "on forum").
  isOnForum?: boolean
  platform: string[]
  language: string[]
  resourceUpdateTime: Date | string
  // U1: optional on card; nil = unknown.
  releaseDate?: string | null
  releaseDateTBA?: boolean
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
