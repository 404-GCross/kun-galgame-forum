// Single source of truth for notification-category keys + labels (BE migration
// 053). Shared by the preference editor (message/NotificationPreference.vue)
// and the muted-messages view (pages/message/muted.vue) so labels never drift.

export interface NotificationCategory {
  key: string
  label: string
}

export interface NotificationCategoryGroup {
  value: string
  textValue: string
  icon: string
  // `local` categories live in the `message` table — they show up in the
  // notification list and the muted view. `chat` / `wiki` are separate streams.
  stream: 'local' | 'chat' | 'wiki'
  items: NotificationCategory[]
}

// Keys mirror the server whitelist (prefs.go). Order = tab order in the editor.
export const notificationCategoryGroups: NotificationCategoryGroup[] = [
  {
    value: 'interaction',
    textValue: '互动',
    icon: 'lucide:heart',
    stream: 'local',
    items: [
      { key: 'upvoted', label: '被推荐' },
      { key: 'liked', label: '被点赞' },
      { key: 'favorite', label: '被收藏' },
      { key: 'mentioned', label: '被 @ 提及' }
    ]
  },
  {
    value: 'reply',
    textValue: '回复评论',
    icon: 'lucide:message-circle',
    stream: 'local',
    items: [
      { key: 'replied', label: '收到回复' },
      { key: 'commented', label: '收到评论' },
      { key: 'solution', label: '回复被采纳为最佳答案' },
      { key: 'pin-reply', label: '回复被置顶' },
      { key: 'quiz-answered', label: '题目被回答' }
    ]
  },
  {
    value: 'review',
    textValue: '内容审核',
    icon: 'lucide:git-pull-request',
    stream: 'local',
    items: [
      { key: 'requested', label: '收到更新请求' },
      { key: 'merged', label: '更新请求被合并' },
      { key: 'declined', label: '更新请求被拒绝' },
      { key: 'expired', label: '资源链接被报告过期' }
    ]
  },
  {
    value: 'chat',
    textValue: '私信',
    icon: 'lucide:mail',
    stream: 'chat',
    items: [{ key: 'chat', label: '私信消息' }]
  },
  {
    value: 'wiki',
    textValue: '资料库',
    icon: 'lucide:book-open',
    stream: 'wiki',
    items: [
      { key: 'wiki:approved', label: '资料库编辑通过' },
      { key: 'wiki:declined', label: '资料库编辑被拒' },
      { key: 'wiki:banned', label: '资料库被封禁' },
      { key: 'wiki:unbanned', label: '资料库被解封' }
    ]
  }
]

// Local message.type categories only — the ones that appear in the notification
// list / muted view (chat + wiki are separate streams).
export const localNotificationCategories: NotificationCategory[] =
  notificationCategoryGroups
    .filter((g) => g.stream === 'local')
    .flatMap((g) => g.items)

// Flat key → label lookup across every stream.
export const notificationCategoryLabel: Record<string, string> =
  Object.fromEntries(
    notificationCategoryGroups.flatMap((g) =>
      g.items.map((i) => [i.key, i.label] as const)
    )
  )
