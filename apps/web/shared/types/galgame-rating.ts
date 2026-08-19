import type { GalgameOfficialItem } from './galgame-official'

export interface GalgameRatingGalgameInfo {
  id: number
  name: string
  name_original: string
  content_limit: string
  official: GalgameOfficialItem[]
  age_limit: string
  original_language: string
  effective_banner_hash?: string
  effective_banner_url?: string
  effective_banner_width?: number
  effective_banner_height?: number
  effective_banner_thumbhash?: string
  rating: number
  rating_count: number
}

export interface GalgameRatingCard {
  id: number
  user: KunUser
  recommend: string
  overall: number
  view: number
  galgame_type: string[]
  play_status: string
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

  like_count: number
  created: Date | string
  updated: Date | string

  galgame: {
    id: number
    name: string
    content_limit: string
  }
}

export interface GalgameRatingDetails extends GalgameRatingCard {
  is_liked: boolean
  liked_users: KunUser[]
  galgame: GalgameRatingGalgameInfo
}

export interface GalgameRatingCardOnGalgamePage extends GalgameRatingCard {
  is_liked: boolean
}
