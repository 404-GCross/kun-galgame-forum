export type FriendLinkCategory = 'official' | 'galgame' | 'others'

export type FriendLinkStatus = 'normal' | 'essential' | 'down'

export interface FriendLink {
  id: number
  category: FriendLinkCategory
  name: string
  link: string
  description: string
  /** Legacy full image URL (image_service webp, or the legacy /friends/<name>.webp). */
  banner: string
  /** Content-addressed image hash driving the new uploader. */
  bannerImageHash: string
  /** Resolved CDN url for display (falls back to the legacy `banner`). */
  bannerUrl: string
  status: FriendLinkStatus
  sortOrder: number
  created: string
  updated: string
}

/** Response shape of GET /friend-link — links grouped by the 3 fixed categories. */
export interface GroupedFriendLinks {
  official: FriendLink[]
  galgame: FriendLink[]
  others: FriendLink[]
}

/** Create/update payload from the admin form (id present only when editing). */
export interface FriendLinkInput {
  id?: number
  category: FriendLinkCategory
  name: string
  link: string
  description: string
  /** Legacy URL, kept and submitted unchanged so un-migrated rows aren't wiped. */
  banner: string
  /** Content-addressed image hash from the cover uploader. */
  bannerImageHash: string
  status: FriendLinkStatus
}
