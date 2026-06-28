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

export const KUN_USER_PAGE_NAV_MAP: Record<string, string> = {
  profile: '个人信息',
  setting: '信息设置',
  topic: '话题',
  galgame: 'Galgame',
  rating: 'Gal 评分',
  resource: 'Gal 资源',
  publish: '已发布',
  like: '已点赞',
  favorite: '已收藏',
  upvote: '已推',
  contribute: '已贡献',
  reply: '回复',
  comment: '评论'
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
