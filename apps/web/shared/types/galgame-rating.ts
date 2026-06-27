import type { GalgameOfficialItem } from './galgame-official'
import type { GalgameSeries } from './galgame-series'

export interface GalgameRatingGalgameInfo {
  id: number
  name: KunLanguage
  contentLimit: string
  official: GalgameOfficialItem[]
  ageLimit: string
  originalLanguage: string
  banner: string
  effective_banner_hash?: string
  effective_banner_url?: string
  // Derived banner's intrinsic metadata for no-CLS aspect-ratio + blur-up.
  effective_banner_width?: number
  effective_banner_height?: number
  effective_banner_thumbhash?: string
  // rating overall average
  rating: number
  ratingCount: number
}

export interface GalgameRatingCard {
  id: number
  user: KunUser
  recommend: string
  overall: number
  view: number
  galgameType: string[]
  play_status: string
  // Author's short writeup; BE returns it on every list row (RatingCard),
  // so listing it here mirrors the wire shape. Card.vue doesn't render it
  // today but consumers like user/Rating.vue may surface a preview later
  // without having to widen the type.
  short_summary: string

  art: number
  story: number
  music: number
  character: number
  route: number
  system: number
  voice: number
  replay_value: number
  spoiler_level: string

  likeCount: number
  created: Date | string
  updated: Date | string

  galgame: {
    id: number
    name: KunLanguage
    contentLimit: string
  }
}

export interface GalgameRatingComment {
  id: number
  content: string
  user: KunUser
  targetUser: KunUser | null

  created: Date | string
  updated: Date | string
}

export interface GalgameRatingDetails extends GalgameRatingCard {
  isLiked: boolean
  likedUsers: KunUser[]
  comments: GalgameRatingComment[]
  galgame: GalgameRatingGalgameInfo
  galgameSeries: GalgameSeries | null
}

export interface GalgameRatingCardOnGalgamePage extends GalgameRatingCard {
  isLiked: boolean
}
