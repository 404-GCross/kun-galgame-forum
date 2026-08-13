import type { KunGalgameOfficialCategory } from '~/constants/galgameOfficial'
import type { GalgameCard } from './galgame'

export interface GalgameOfficialItem {
  id: number
  name: string
  link: string
  category: KunGalgameOfficialCategory
  roles?: string[]
  lang: string
  alias: string[]
  galgame_count: number
  logo?: string
}

export interface GalgameOfficialLink {
  source: string
  name: string
  url: string
}

export interface GalgameOfficialDetail {
  id: number
  name: string
  original: string
  links: GalgameOfficialLink[]
  link: string
  logo: string
  category: KunGalgameOfficialCategory
  lang: string
  description: string
  alias: string[]
  galgame: GalgameCard[]
  galgame_count: number
  own_galgame_count: number
  imprint_galgame_count: number
  moved_to?: number
}

export type GalgameOfficialRelation =
  | 'parent'
  | 'subsidiary'
  | 'imprint'
  | 'imprint_of'
  | 'spawned'
  | 'origin'
  | 'succeeded_by'
  | 'formerly'

export interface GalgameOfficialRelationNode {
  id: number
  name: string
  logo: string
  work_count: number
}

export interface GalgameOfficialRelationEdge {
  from: number
  to: number
  relation: GalgameOfficialRelation
}

export interface GalgameOfficialRelationGraph {
  nodes: GalgameOfficialRelationNode[]
  edges: GalgameOfficialRelationEdge[]
}
