export interface GalgameStaffWork extends GalgameCard {
  catalog_id: number
  roles: string[]
  characters?: string[]
}

export interface GalgameStaffLink {
  source: string
  name: string
  url?: string
}

export interface GalgameStaffSibling {
  id: number
  name: string
}

export interface GalgameStaffDetail {
  id: number
  name: string
  name_original?: string
  latin?: string
  intro: string
  intro_machine: boolean
  photo?: string
  gender?: number | null
  birth_y?: number | null
  birth_m?: number | null
  birth_d?: number | null
  links: GalgameStaffLink[]
  siblings: GalgameStaffSibling[]
  roles: string[]
  works: GalgameStaffWork[]
  next_offset: number | null
  moved_to?: number
}
