<script setup lang="ts">
// Notification-category preferences (BE migration 053). Each switch is
// "接收该类通知" — ON = receive, OFF = mute. Muting only stops the category from
// driving the top-bar red dot / unread badges; the messages are still kept and
// stay visible in the notification center. The catalog keys mirror the server
// whitelist: message.type values, the "system"/"chat" pseudo keys, and
// namespaced "wiki:*" keys.
interface CategoryItem {
  key: string
  label: string
}
interface CategoryGroup {
  title: string
  items: CategoryItem[]
}

const groups: CategoryGroup[] = [
  {
    title: '互动',
    items: [
      { key: 'upvoted', label: '被推荐' },
      { key: 'liked', label: '被点赞' },
      { key: 'favorite', label: '被收藏' }
    ]
  },
  {
    title: '回复与评论',
    items: [
      { key: 'replied', label: '收到回复' },
      { key: 'commented', label: '收到评论' },
      { key: 'solution', label: '回复被采纳为最佳答案' },
      { key: 'pin-reply', label: '回复被置顶' }
    ]
  },
  {
    title: '提及',
    items: [{ key: 'mentioned', label: '被 @ 提及' }]
  },
  {
    title: '题库',
    items: [{ key: 'quiz-answered', label: '题目被回答' }]
  },
  {
    title: '内容与资源审核',
    items: [
      { key: 'requested', label: '收到更新请求' },
      { key: 'merged', label: '更新请求被合并' },
      { key: 'declined', label: '更新请求被拒绝' },
      { key: 'expired', label: '资源链接被报告过期' }
    ]
  },
  {
    title: '系统公告',
    items: [{ key: 'system', label: '官方系统公告' }]
  },
  {
    title: '私信',
    items: [{ key: 'chat', label: '私信消息' }]
  },
  {
    title: 'Wiki 审核反馈',
    items: [
      { key: 'wiki:approved', label: 'Wiki 编辑通过' },
      { key: 'wiki:declined', label: 'Wiki 编辑被拒' },
      { key: 'wiki:banned', label: 'Wiki 被封禁' },
      { key: 'wiki:unbanned', label: 'Wiki 被解封' }
    ]
  }
]

const allKeys = groups.flatMap((g) => g.items.map((i) => i.key))

// enabled[key] === true → receiving (not muted). Defaults to true.
const enabled = reactive<Record<string, boolean>>(
  Object.fromEntries(allKeys.map((k) => [k, true]))
)
const isLoading = ref(true)

const applyMuted = (muted: string[]) => {
  const mutedSet = new Set(muted)
  for (const key of allKeys) {
    enabled[key] = !mutedSet.has(key)
  }
}

const collectMuted = () => allKeys.filter((key) => !enabled[key])

const onToggle = async (key: string, value: boolean) => {
  const previous = enabled[key] ?? true
  enabled[key] = value

  const result = await kunFetch<NotificationPreference>(
    '/user/notification-preferences',
    { method: 'PUT', body: { muted_types: collectMuted() } }
  )

  if (result?.muted_types) {
    // Trust the sanitized set the server echoes back.
    applyMuted(result.muted_types)
  } else {
    enabled[key] = previous
    useMessage('保存失败，请稍后重试', 'error')
  }
}

onMounted(async () => {
  const result = await kunFetch<NotificationPreference>(
    '/user/notification-preferences'
  )
  if (result?.muted_types) {
    applyMuted(result.muted_types)
  }
  isLoading.value = false
})
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-4">
    <div>
      <span class="text-xl">消息通知</span>
      <p class="text-default-500 text-sm">
        关闭某类通知后，它不会再让顶栏红点闪烁或点亮未读角标，但消息仍会保留在通知中心里，你随时可以进去查看。
      </p>
    </div>

    <div v-for="group in groups" :key="group.title" class="space-y-2">
      <span class="text-default-600 text-sm font-medium">{{ group.title }}</span>
      <div class="divide-default-100 divide-y">
        <div
          v-for="item in group.items"
          :key="item.key"
          class="flex items-center justify-between py-2"
        >
          <span class="text-default-700 text-sm">{{ item.label }}</span>
          <KunSwitch
            :model-value="enabled[item.key] ?? true"
            :disabled="isLoading"
            @update:model-value="(v: boolean) => onToggle(item.key, v)"
          />
        </div>
      </div>
    </div>
  </KunCard>
</template>
