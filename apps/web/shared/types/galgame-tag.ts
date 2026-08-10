import type { KunGalgameTagCategory } from '~/constants/galgameTag'
import type { GalgameCard } from './galgame'

export interface GalgameTag {
  id: number
  name: string
  category: KunGalgameTagCategory
}

export interface GalgameTagItem {
  id: number
  name: string
  category: KunGalgameTagCategory
  galgame_count: number
}

export interface GalgameTaxonomySearchItem {
  id: number
  name: string
  logo?: string
}

export interface GalgameTagDetail {
  id: number
  name: string
  category: KunGalgameTagCategory
  hidden: boolean
  description: string
  alias: string[]
  galgame: GalgameCard[]
  galgame_count: number
}
