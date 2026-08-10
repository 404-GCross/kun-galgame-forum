import type {
  FriendLinkCategory,
  FriendLinkStatus
} from '../../shared/types/friend-link'

export const FRIEND_LINK_CATEGORIES: {
  key: FriendLinkCategory
  label: string
}[] = [
  { key: 'official', label: '官方网站' },
  { key: 'galgame', label: 'Galgame 网站' },
  { key: 'others', label: '其它网站' }
]

export const FRIEND_LINK_CATEGORY_OPTIONS = FRIEND_LINK_CATEGORIES.map((c) => ({
  value: c.key,
  label: c.label
}))

export const FRIEND_LINK_STATUS_OPTIONS = [
  { value: 'normal', label: '正常' },
  { value: 'essential', label: '精选' },
  { value: 'down', label: '已下线' }
] as const

export const FRIEND_LINK_STATUS_CHIP: Record<
  FriendLinkStatus,
  { label: string; color: 'primary' | 'danger' } | null
> = {
  normal: null,
  essential: { label: '精选', color: 'primary' },
  down: { label: '已下线', color: 'danger' }
}
