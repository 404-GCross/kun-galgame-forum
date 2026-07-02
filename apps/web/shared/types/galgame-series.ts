import type { GalgameCard } from './galgame.d.ts'

export interface GalgameSeriesSample {
  name: KunLanguage
  banner: string
  effective_banner_hash?: string
  effective_banner_url?: string
}

export interface GalgameSeries {
  id: number
  name: string
  description: string
  is_nsfw: boolean
  sample_galgame: GalgameSeriesSample[]
  galgame_count: number
  created: Date | string
  updated: Date | string
}

export interface GalgameSeriesDetail extends GalgameSeries {
  description: string
  galgame: GalgameCard[]
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
