import type {
  GalgameArtMeta,
  GalgameCard,
  GalgameDetailCharacterVoice
} from './galgame'

export interface GalgameCharacterIntro {
  lang: string
  intro: string
  source: string
  machine: boolean
}

export interface GalgameCharacterTrait {
  id: number
  name: string
  group: string
  spoiler: number
  lie: boolean
}

export interface GalgameCharacterLink {
  source: string
  name: string
  url?: string
}

export interface GalgameCharacterWork extends GalgameCard {
  catalog_id: number
  voices: GalgameDetailCharacterVoice[]
}

export interface GalgameCharacterDetail {
  id: number
  name: string
  name_ja?: string
  name_zh?: string
  latin?: string
  image: string
  figure: string
  image_meta?: GalgameArtMeta
  figure_meta?: GalgameArtMeta
  intro: string
  intros: GalgameCharacterIntro[]
  traits: GalgameCharacterTrait[]
  links: GalgameCharacterLink[]
  works: GalgameCharacterWork[]
  next_offset: number | null
  moved_to?: number
}
