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

export interface GalgameTagDetail {
  id: number
  name: string
  category: KunGalgameTagCategory
  description: string
  alias: string[]
  galgame: GalgameCard[]
  galgame_count: number
}
