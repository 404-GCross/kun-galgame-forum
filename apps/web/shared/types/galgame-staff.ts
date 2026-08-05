// The 制作人员 page — one CREDITED NAME from the catalog, not one human.
//
// The registry links a name to a person only where the evidence supports it and
// the link is public, so `siblings` empty means "no other pen name published",
// never "this person has none". The page says as much rather than implying the
// name is the whole person.

/** One entry in a credited name's filmography. */
export interface GalgameStaffWork {
  /** Forum gid, or 0 when this game is not on the forum — most of a working
   *  career is games the forum has never ingested. Those cards do not link. */
  id: number
  catalog_id: number
  name: KunLanguage
  banner: string
  banner_width?: number
  banner_height?: number
  banner_thumbhash?: string
  content_limit: string
  release_date: string | null
  /** This person's credits on THIS game — the reason a filmography beats a grid. */
  roles: string[]
  /** Voice acting only: the characters this person voiced in this game. */
  characters?: string[]
}

/** An external identity page. `url` is absent for a source with no verified
 *  person-page template; the row then renders as plain text. */
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
  name_ja?: string
  name_zh?: string
  latin?: string
  intro: string
  links: GalgameStaffLink[]
  siblings: GalgameStaffSibling[]
  /** Positions seen on the loaded works, ordered authorship-first and
   *  deliberately WITHOUT counts — the credits list is offset-paged and the
   *  catalog publishes no total, so any number would describe one page. */
  roles: string[]
  works: GalgameStaffWork[]
  /** null on the last page — the only end-of-list signal there is. */
  next_offset: number | null
}
