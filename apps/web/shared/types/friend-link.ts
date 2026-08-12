export type FriendLinkCategory = 'official' | 'galgame' | 'others'

export type FriendLinkStatus = 'normal' | 'essential' | 'down'

export interface FriendLink {
  id: number
  category: FriendLinkCategory
  name: string
  link: string
  description: string
  banner: string
  banner_image_hash: string
  banner_url: string
  status: FriendLinkStatus
  sort_order: number
  created: string
  updated: string
}

export interface GroupedFriendLinks {
  official: FriendLink[]
  galgame: FriendLink[]
  others: FriendLink[]
}

export interface FriendLinkInput {
  id?: number
  category: FriendLinkCategory
  name: string
  link: string
  description: string
  banner: string
  banner_image_hash: string
  status: FriendLinkStatus
}
