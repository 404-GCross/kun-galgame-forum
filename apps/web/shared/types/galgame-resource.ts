export interface GalgameResource {
  id: number
  view: number
  galgame_id: number
  user: KunUser
  type: string
  language: string
  platform: string
  size: string
  status: number
  download: number
  like_count: number
  is_liked: boolean
  // Comment counter (migration 065), maintained ±1 by the community comment BFF
  // — a tolerated display counter, not a source of truth.
  comment_count: number
  link_domain: string
  /**
   * Pre-computed display labels for the resource's hosting providers
   * (e.g. ["百度网盘", "OneDrive"]). Resolved by the backend at write time
   * — do not re-derive from `linkDomain` in the UI.
   */
  provider_names: string[]
  note: string
  /**
   * Server-rendered, sanitized HTML of `note` (Markdown → HTML via the shared
   * markdown.Render pipeline, same as topic/comment content). Render this with
   * `<KunContent>`; keep `note` (raw markdown) only for re-seeding the editor.
   */
  note_html: string
  created: Date | string
  edited: Date | string | null
}

export interface GalgameResourceDetailLink extends GalgameResource {
  link: string[]
  code: string
  note: string
  password: string
}

export interface GalgameResourceCard extends GalgameResource {
  galgame_name: KunLanguage
}

export interface GalgameResourceSummary {
  id: number
  name: KunLanguage
  banner: string
  effective_banner_hash?: string
  effective_banner_url?: string
  // Derived banner's intrinsic metadata for no-CLS aspect-ratio + blur-up.
  effective_banner_width?: number
  effective_banner_height?: number
  effective_banner_thumbhash?: string
  content_limit: string
  resource_update_time: Date | string
  view: number
  original_language: string
  age_limit: KunAgeLimit
  platform: string[]
  language: string[]
  type: string[]
}

export interface GalgameResourcePageData {
  galgame: GalgameResourceSummary
  resource: GalgameResource
  recommendations: GalgameResourceCard[]
}
