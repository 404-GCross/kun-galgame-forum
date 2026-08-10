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
  banner: string
  banner_image_hash: string
  banner_url: string
  status: number
  is_pin: boolean
  view: number
  published_time: Date | string
  edited_time: Date | string | null
  content_markdown: string
  category_id: number
  author_id: number
  category: DocArticleCategoryBrief
  tag_ids?: number[]
  created: Date | string
  updated: Date | string
  content_html?: string
  toc?: DocTocLink[]
}

export type DocArticleSummary = DocArticle
export type DocArticleDetail = DocArticle

export interface DocArticleListResponse {
  items: DocArticle[]
  total: number
}
