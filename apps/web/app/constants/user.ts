import type { KunTabItem } from '@kungal/ui-vue'

type _NavItemData = { text: string; path: string }

export const TOPIC_NAV_CONFIG: Record<
  (typeof KUN_USER_PAGE_TOPIC_TYPE)[number],
  _NavItemData
> = {
  topic: { text: '已发布', path: 'topic' },
  topic_like: { text: '已点赞', path: 'topic-like' },
  topic_upvote: { text: '已推', path: 'topic-upvote' },
  topic_favorite: { text: '已收藏', path: 'topic-favorite' },
  topic_hide: { text: '已隐藏', path: 'topic-hide' }
}

export const REPLY_NAV_CONFIG: Record<
  (typeof KUN_USER_PAGE_REPLY_TYPE)[number],
  _NavItemData
> = {
  reply_created: { text: '已发布', path: 'reply-created' },
  reply_target: { text: '被回复', path: 'reply-target' },
  reply_like: { text: '已点赞', path: 'reply-like' }
}

export const COMMENT_NAV_CONFIG: Record<
  (typeof KUN_USER_PAGE_COMMENT_TYPE)[number],
  _NavItemData
> = {
  comment_created: { text: '已发布', path: 'comment-created' },
  comment_target: { text: '被评论', path: 'comment-target' },
  comment_like: { text: '已点赞', path: 'comment-like' }
}

export const GALGAME_NAV_CONFIG: Record<
  (typeof KUN_USER_PAGE_GALGAME_TYPE)[number],
  _NavItemData
> = {
  galgame_publish: { text: '已发布', path: 'galgame-publish' },
  galgame_contributed: { text: '贡献的', path: 'galgame-contributed' },
  galgame_like: { text: '已点赞', path: 'galgame-like' },
  galgame_favorite: { text: '已收藏', path: 'galgame-favorite' },
  galgame_comment: { text: '评论', path: 'galgame-comment' },
  galgame_comment_target: { text: '被评论', path: 'galgame-comment-target' },
  galgame_comment_like: { text: '点赞评论', path: 'galgame-comment-like' }
}

export const GALGAME_RESOURCE_NAV_CONFIG: Record<
  (typeof KUN_USER_PAGE_GALGAME_RESOURCE_TYPE)[number],
  _NavItemData
> = {
  valid: { text: '有效资源', path: 'valid' },
  expire: { text: '过期资源', path: 'expire' },
  galgame_resource_like: { text: '点赞资源', path: 'galgame-resource-like' }
}

const createUserPageNavItems = <T extends string>(
  userId: number,
  basePath: string,
  typesArray: readonly T[],
  config: Record<T, _NavItemData>
): KunTabItem[] => {
  return typesArray.map((value) => ({
    value,
    textValue: config[value].text,
    href: `/user/${userId}/${basePath}/${config[value].path}`
  }))
}

export const kunUserTopicNavItem = (userId: number): KunTabItem[] => {
  return createUserPageNavItems(
    userId,
    'topic',
    KUN_USER_PAGE_TOPIC_TYPE,
    TOPIC_NAV_CONFIG
  )
}

export const kunUserReplyNavItem = (userId: number): KunTabItem[] => {
  return createUserPageNavItems(
    userId,
    'reply',
    KUN_USER_PAGE_REPLY_TYPE,
    REPLY_NAV_CONFIG
  )
}

export const kunUserCommentNavItem = (userId: number): KunTabItem[] => {
  return createUserPageNavItems(
    userId,
    'comment',
    KUN_USER_PAGE_COMMENT_TYPE,
    COMMENT_NAV_CONFIG
  )
}

export const kunUserGalgameNavItem = (userId: number): KunTabItem[] => {
  return createUserPageNavItems(
    userId,
    'galgame',
    KUN_USER_PAGE_GALGAME_TYPE,
    GALGAME_NAV_CONFIG
  )
}

export const kunUserGalgameResourceNavItem = (userId: number): KunTabItem[] => {
  return createUserPageNavItems(
    userId,
    'resource',
    KUN_USER_PAGE_GALGAME_RESOURCE_TYPE,
    GALGAME_RESOURCE_NAV_CONFIG
  )
}

export const KUN_USER_PAGE_TOPIC_TYPE = [
  'topic',
  'topic_like',
  'topic_upvote',
  'topic_favorite',
  'topic_hide'
] as const

export const KUN_USER_PAGE_REPLY_TYPE = [
  'reply_created',
  'reply_target',
  'reply_like'
] as const

export const KUN_USER_PAGE_COMMENT_TYPE = [
  'comment_created',
  'comment_target',
  'comment_like'
] as const

export const KUN_USER_PAGE_GALGAME_TYPE = [
  'galgame_publish',
  'galgame_contributed',
  'galgame_like',
  'galgame_favorite',
  'galgame_comment',
  'galgame_comment_target',
  'galgame_comment_like'
] as const

export const KUN_USER_PAGE_GALGAME_RESOURCE_TYPE = [
  'valid',
  'expire',
  'galgame_resource_like'
] as const

// The profile's TOP-LEVEL tab strip (the horizontal bar under the header).
// 动态 (activity) is the landing tab; 关于 (info) holds the full stats panel;
// 设置 is owner-only. `value` is the 3rd URL segment so the active tab can be
// derived from the route; tabs with sub-filters point at their default child.
export const kunUserMainNav = (
  userId: number,
  isOwner: boolean
): KunTabItem[] => {
  const base = `/user/${userId}`
  const items: KunTabItem[] = [
    { value: 'activity', textValue: '动态', href: `${base}/activity`, icon: 'lucide:activity' },
    { value: 'topic', textValue: '话题', href: `${base}/topic/topic`, icon: 'lucide:square-gantt-chart' },
    { value: 'galgame', textValue: 'Galgame', href: `${base}/galgame/galgame-publish`, icon: 'lucide:gamepad-2' },
    { value: 'rating', textValue: 'Gal 评分', href: `${base}/rating`, icon: 'lucide:star' },
    { value: 'resource', textValue: 'Gal 资源', href: `${base}/resource/valid`, icon: 'lucide:package' },
    { value: 'reply', textValue: '回复', href: `${base}/reply/reply-created`, icon: 'carbon:reply' },
    { value: 'comment', textValue: '评论', href: `${base}/comment/comment-created`, icon: 'uil:comment-dots' },
    { value: 'info', textValue: '关于', href: `${base}/info`, icon: 'lucide:user-round' }
  ]
  if (isOwner) {
    items.push({ value: 'setting', textValue: '设置', href: `${base}/setting`, icon: 'lucide:settings' })
  }
  return items
}

// OAuth named roles → display label (docs/oauth/11-roles.md). `user` is implicit
// (an empty role set) and has no entry — see managementRoleLabel's 普通用户 fallback.
export const KUN_USER_ROLE_MAP: Record<string, string> = {
  ren: '莲',
  admin: '管理员',
  moderator: '版主',
  creator: '创作者'
}

// The single management-role label for a user's role set, picking the highest of
// the inclusive management axis (ren > admin > moderator). `creator` is orthogonal
// and rendered as its own chip, so it's excluded here; an empty set is 普通用户.
export const managementRoleLabel = (roles: string[]): string =>
  roles.includes('ren')
    ? KUN_USER_ROLE_MAP.ren!
    : roles.includes('admin')
      ? KUN_USER_ROLE_MAP.admin!
      : roles.includes('moderator')
        ? KUN_USER_ROLE_MAP.moderator!
        : '普通用户'

export const KUN_USER_STATUS_MAP: Record<number, string> = {
  0: '正常',
  1: '封禁'
}
