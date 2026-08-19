export interface RankingUserItem {
  id: number
  name: string
  avatar: string
  bio: string
  sort_field: string
  value: number
}

export interface RankingTopicItem {
  id: number
  title: string
  user: KunUser
  sort_field: string
  value: number
}

export interface RankingGalgameItem {
  id: number
  name: string
  user: KunUser
  effective_banner_hash?: string
  effective_banner_url?: string
  sort_field: string
  value: number
}
