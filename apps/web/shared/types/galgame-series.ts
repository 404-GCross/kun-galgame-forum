import type { GalgameCard } from './galgame'

export interface GalgameSeriesItem {
  id: number
  name: string
  galgame_count: number
}

export interface GalgameSeriesSample {
  name: KunLanguage
  effective_banner_hash?: string
  effective_banner_url?: string
  effective_banner_thumbhash?: string
}

export interface GalgameSeriesCard {
  id: number
  name: string
  is_nsfw: boolean
  galgame_count: number
  sample_galgame: GalgameSeriesSample[]
}

export interface GalgameSeriesDetail {
  id: number
  name: string
  description: string
  galgame: GalgameCard[]
  galgame_count: number
  unpublished_galgame: GalgameCard[]
}

export interface GalgameSeriesSearchItem {
  id: number
  name_en_us?: string
  name_ja_jp?: string
  name_zh_cn?: string
  name_zh_tw?: string
}

export interface GalgameDetailSeriesRef {
  id: number
  name: string
}
