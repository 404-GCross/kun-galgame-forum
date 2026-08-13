import type { GalgameCard } from './galgame'

export interface GalgameEngineItem {
  id: number
  name: string
  description?: string
  alias: string[]
  galgame_count: number
}

export interface GalgameEngineDetail {
  id: number
  name: string
  description: string
  alias: string[]
  galgame: GalgameCard[]
  galgame_count: number
}
