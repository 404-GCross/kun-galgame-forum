export interface GalgameResource {
  id: number
  view: number
  galgameId: number
  user: KunUser
  type: string
  language: string
  platform: string
  size: string
  status: number
  download: number
  likeCount: number
  isLiked: boolean
  linkDomain: string
  /**
   * Pre-computed display labels for the resource's hosting providers
   * (e.g. ["百度网盘", "OneDrive"]). Resolved by the backend at write time
   * — do not re-derive from `linkDomain` in the UI.
   */
  providerNames: string[]
  note: string
  /**
   * Server-rendered, sanitized HTML of `note` (Markdown → HTML via the shared
   * markdown.Render pipeline, same as topic/comment content). Render this with
   * `<KunContent>`; keep `note` (raw markdown) only for re-seeding the editor.
   */
  noteHtml: string
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
  galgameName: KunLanguage
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
  contentLimit: string
  resourceUpdateTime: Date | string
  view: number
  originalLanguage: string
  ageLimit: KunAgeLimit
  platform: string[]
  language: string[]
  type: string[]
}

export interface GalgameResourcePageData {
  galgame: GalgameResourceSummary
  resource: GalgameResource
  recommendations: GalgameResourceCard[]
}
