export interface KunNewsPublisher {
  id: number
  name: string
  avatar: string
}

export interface KunNewsSource {
  key: string
  name: string
  homepage_url: string
  column_url: string
  attribution: string
  publisher: KunNewsPublisher | null
}

export interface KunNewsItem {
  id: number
  source_key: string
  lane: 'news' | 'column'
  title: string
  preview: string
  source_url: string
  banner_url: string
  published_at: string
}

export interface KunNewsFeed {
  items: KunNewsItem[]
  sources: Record<string, KunNewsSource>
  count: number
  next_cursor: string
}
