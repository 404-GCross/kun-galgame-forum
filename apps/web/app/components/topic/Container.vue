<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'

// /topic list — page-based (URL ?page=), 50 per page, jump to top on page change.
const route = useRoute()

// Page lives in the URL so the view is shareable + survives back/forward; the
// default 1 is omitted from the URL. Number transform keeps the ref numeric.
const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const limit = 50

const { data, status, refresh } = await useKunFetch<{
  topics: TopicCard[]
  total: number
}>('/topic', {
  method: 'GET',
  query: {
    page,
    limit,
    sort_field: 'status_update_time',
    sort_order: 'desc',
    category: 'all'
  },
  // Don't auto-refetch on every query-ref change: opening a topic navigates to
  // /topic/:id, which resets the page ref and would otherwise fire a wasted
  // fetch right before unmount. Refetch manually, ONLY while still on this list.
  watch: false
})

const listPath = route.path
watch(
  () => route.fullPath,
  () => {
    if (route.path !== listPath) return
    refresh()
    // 翻页后回到顶部
    if (import.meta.client) window.scrollTo({ top: 0, behavior: 'smooth' })
  }
)
</script>

<template>
  <div class="flex flex-col gap-6">
    <KunHeader
      name="话题列表"
      description="鲲 Galgame 论坛的全部话题，涵盖 Galgame 讨论、技术交流、资源求助与日常闲聊，在这里和大家一起畅所欲言。"
    />

    <template v-if="data">
      <!-- List layout: each topic separated by a faint divider, no card chrome. -->
      <KunLoading :loading="status === 'pending'">
        <div class="divide-default-200/60 divide-y">
          <TopicCard
            v-for="topic in data.topics"
            :key="topic.id"
            :topic="topic"
          />
        </div>
      </KunLoading>

      <KunNull
        v-if="!data.topics.length"
        description="真的一滴也不剩了呜呜呜"
      />

      <div class="flex justify-center pt-4">
        <KunPagination
          v-model:current-page="page"
          :total-page="Math.ceil(data.total / limit)"
          :is-loading="status === 'pending'"
        />
      </div>
    </template>
  </div>
</template>
