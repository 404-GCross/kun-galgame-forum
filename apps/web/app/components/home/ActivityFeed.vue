<script setup lang="ts">
import { useIntersectionObserver, useThrottleFn } from '@vueuse/core'

const props = defineProps<{ tabId: string; types: string }>()

const settings = usePersistSettingsStore()

const items = ref<ActivityItem[]>([])
const cursor = ref('')
const hasMore = ref(true)
const isLoadingMore = ref(false)

const MAX_AUTO_LOADS = 4
const autoLoadCount = ref(0)
let controller: AbortController | null = null

const { data, status } = await useKunFetch<{
  items: ActivityItem[]
  next_cursor: string
}>('/activity/tab', {
  method: 'GET',
  query: computed(() => ({
    types: props.types,
    limit: 30,
    show_no_resource: settings.showKUNGalgameNoResource,
    force_sfw: props.tabId === 'all'
  }))
})

watch(
  data,
  (page) => {
    if (!page) return
    items.value = page.items
    cursor.value = page.next_cursor
    hasMore.value = !!page.next_cursor
  },
  { immediate: true }
)

watch(
  () => props.tabId,
  () => {
    cursor.value = ''
    hasMore.value = true
    autoLoadCount.value = 0
    controller?.abort()
  }
)

const loadMore = async (auto = false) => {
  if (isLoadingMore.value || !hasMore.value || !cursor.value) return
  if (auto) {
    if (autoLoadCount.value >= MAX_AUTO_LOADS) return
    autoLoadCount.value++
  } else {
    autoLoadCount.value = 0
  }
  const tab = props.tabId
  const types = props.types
  isLoadingMore.value = true
  controller = new AbortController()
  const next = await kunFetch<{ items: ActivityItem[]; next_cursor: string }>(
    '/activity/tab',
    {
      method: 'GET',
      query: {
        types,
        limit: 30,
        cursor: cursor.value,
        show_no_resource: settings.showKUNGalgameNoResource,
        force_sfw: tab === 'all'
      },
      signal: controller.signal
    }
  )
  isLoadingMore.value = false
  if (!next || props.tabId !== tab) return
  items.value.push(...next.items)
  cursor.value = next.next_cursor
  hasMore.value = !!next.next_cursor
}

const autoLoad = useThrottleFn(() => loadMore(true), 600)
onBeforeUnmount(() => controller?.abort())

const sentinel = ref<HTMLElement | null>(null)
useIntersectionObserver(
  sentinel,
  ([entry]) => {
    if (entry?.isIntersecting) autoLoad()
  },
  { rootMargin: '150px' }
)
</script>

<template>
  <KunLoadingDim
    class="min-w-0"
    :loading="status === 'pending' && items.length > 0"
  >
    <KunNull
      v-if="status !== 'pending' && !items.length"
      description="暂无动态"
    />

    <div v-else class="divide-default-200/60 divide-y">
      <div
        v-for="activity in items"
        :key="activity.unique_id"
        class="py-5 first:pt-0 last:pb-0"
      >
        <ActivityCard :activity="activity" />
      </div>
      <template v-if="isLoadingMore">
        <div v-for="n in 3" :key="`skeleton-${n}`" class="py-5">
          <ActivityCardSkeleton />
        </div>
      </template>
    </div>

    <div v-if="items.length" ref="sentinel" class="flex justify-center pt-4">
      <KunButton
        v-if="hasMore && !isLoadingMore"
        variant="light"
        @click="loadMore(false)"
      >
        加载更多
      </KunButton>
      <span v-else-if="!hasMore" class="text-default-400 text-sm">
        没有更多动态了
      </span>
    </div>
  </KunLoadingDim>
</template>
