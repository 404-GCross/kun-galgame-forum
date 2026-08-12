<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import {
  topicListSortFieldOptions,
  type TopicListSortField
} from '~/constants/topic'

const route = useRoute()

const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const sortField = useRouteQuery<TopicListSortField>(
  'sort_field',
  'status_update_time',
  { mode: 'replace' }
)
const sortOrder = useRouteQuery<'asc' | 'desc'>('sort_order', 'desc', {
  mode: 'replace'
})
const limit = 50

const { data, status, refresh } = await useKunFetch<{
  topics: TopicCard[]
  total: number
}>('/topic', {
  method: 'GET',
  query: {
    page,
    limit,
    sort_field: sortField,
    sort_order: sortOrder,
    category: 'all'
  },
  watch: false
})

const setSortField = (
  value: TopicListSortField | TopicListSortField[] | null
) => {
  if (!value || Array.isArray(value)) return
  if (value === sortField.value) return
  page.value = 1
  sortField.value = value
}
const setSortOrder = (value: 'asc' | 'desc') => {
  if (value === sortOrder.value) return
  page.value = 1
  sortOrder.value = value
}

const listPath = route.path
watch(
  () => route.fullPath,
  () => {
    if (route.path !== listPath) return
    refresh()
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

    <div class="flex items-center justify-between gap-3">
      <KunSelect
        class-name="w-44"
        :model-value="sortField"
        :options="topicListSortFieldOptions"
        @update:model-value="setSortField"
      />

      <div class="flex shrink-0 items-center gap-1">
        <KunButton
          :is-icon-only="true"
          :variant="sortOrder === 'desc' ? 'flat' : 'light'"
          color="primary"
          size="sm"
          @click="setSortOrder('desc')"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-down" />
        </KunButton>
        <KunButton
          :is-icon-only="true"
          :variant="sortOrder === 'asc' ? 'flat' : 'light'"
          color="primary"
          size="sm"
          @click="setSortOrder('asc')"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-up" />
        </KunButton>
      </div>
    </div>

    <template v-if="data">
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
