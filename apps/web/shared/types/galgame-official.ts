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
  lang: string
  alias: string[]
  galgame_count: number
}

export interface GalgameOfficialDetail {
  id: number
  name: string
  // Original-language name (wiki PR4 sub-change, K-PR6). BE returns ""
  // when wiki hasn't recorded an original yet. The edit modal pre-fills
  // from this so admins can see the existing value instead of starting
  // from empty.
  original: string
  link: string
  category: KunGalgameOfficialCategory
  lang: string
  description: string
  alias: string[]
  galgame: GalgameCard[]
  galgame_count: number
}
