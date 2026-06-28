export interface DocTocLink {
  id: string
  text: string
  depth: number
  children?: DocTocLink[]
}

export interface DocCategoryItem {
  id: number
  slug: string
  title: string
  description: string
  icon: string
  sort_order: number
  created: Date | string
  updated: Date | string
}

// BE returns the shared response.Paginated envelope `{items, total}`
// (apps/api/pkg/response/response.go). FE consumers already read
// `.items` everywhere, so the legacy `categories` / `page` / `limit`
// keys were dead-and-misleading TS.
export interface DocCategoryListResponse {
  items: DocCategoryItem[]
  total: number
}

export interface DocTagItem {
  id: number
  slug: string
  title: string
  description: string
  created: Date | string
  updated: Date | string
}

export interface DocTagListResponse {
  items: DocTagItem[]
  total: number
}

export interface DocArticleCategoryBrief {
  id: number
  slug: string
  title: string
}

export interface DocArticle {
  id: number
  title: string
  slug: string
  path: string
  description: string
  // Legacy full-URL cover (still written/returned for un-migrated docs).
  banner: string
  // Content-addressed cover: `bannerImageHash` is the stored hash (write +
  // read), `bannerUrl` is the BE-resolved CDN url (falls back to `banner`).
  bannerImageHash: string
  bannerUrl: string
  status: number
  isPin: boolean
  view: number
  publishedTime: Date | string
  editedTime: Date | string | null
  contentMarkdown: string
  categoryId: number
  authorId: number
  category: DocArticleCategoryBrief
  // Embedded by BE so the rewrite flow can pre-fill the tag picker
  // without a second fetch. List endpoints may omit; detail always sets.
  tagIds?: number[]
  created: Date | string
  updated: Date | string
  contentHtml?: string
  toc?: DocTocLink[]
}

// Kept for backward compatibility with components that use the old shape
export type DocArticleSummary = DocArticle
export type DocArticleDetail = DocArticle

export interface DocArticleListResponse {
  items: DocArticle[]
  total: number
}
