export interface WebsiteCategory {
  id: number
  name: string
  label: string
  description: string
}

export interface WebsiteTag {
  id: number
  name: string
  description: string
  label: string
  level: number
}

export interface WebsiteTagDetail {
  id: number
  name: string
  label: string
  level: number
  description: string
  websiteCount: number
  websites: WebsiteCard[]
  created: Date | string
  updated: Date | string
}

export interface WebsiteCategoryDetail {
  id: number
  name: string
  label: string
  description: string
  websiteCount: number
  websites: WebsiteCard[]
  created: Date | string
  updated: Date | string
}

export interface WebsiteCard {
  id: number
  name: string
  description: string
  domain: string
  ageLimit: string
  level: number
  /** Legacy icon URL. */
  icon: string
  /** Content-addressed image hash. */
  iconImageHash: string
  /** Resolved CDN url for display (falls back to the legacy `icon`). */
  iconUrl: string
  price: number
  category: string
}

export interface WebsiteDetail {
  id: number
  name: string
  url: string
  description: string
  /** Legacy icon URL. */
  icon: string
  /** Content-addressed image hash. */
  iconImageHash: string
  /** Resolved CDN url for display (falls back to the legacy `icon`). */
  iconUrl: string
  view: number
  language: string
  ageLimit: 'all' | 'r18'
  category: WebsiteCategory
  tags: WebsiteTag[]
  likeCount: number
  isLiked: boolean
  favoriteCount: number
  isFavorited: boolean
  domain: string[]
  createTime: string
  comment: {
    id: number
    content: string
    user: KunUser
    created: Date | string
    updated: Date | string
  }[]

  created: Date | string
  updated: Date | string
}

export interface WebsiteComment {
  id: number
  websiteId: number
  content: string
  parentId: number | null
  userId: number
  created: Date | string
  edited: Date | string | null
  // Roots carry their full set of descendants flattened here (oldest-first, one
  // visual tier); replies leave it empty. replyCount = reply.length.
  reply: WebsiteComment[]
  replyCount: number
  user: KunUser
  targetUser?: KunUser | null
}
