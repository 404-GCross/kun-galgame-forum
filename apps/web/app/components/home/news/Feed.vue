<script setup lang="ts">
import { useIntersectionObserver, useThrottleFn } from '@vueuse/core'

interface NewsGroup {
  key: string
  date: string
  source: KunNewsSource | undefined
  items: KunNewsItem[]
}

const PAGE_SIZE = 20
const MAX_AUTO_LOADS = 4

const items = ref<KunNewsItem[]>([])
const sources = ref<Record<string, KunNewsSource>>({})
const cursor = ref('')
const hasMore = ref(true)
const isLoadingMore = ref(false)
const autoLoadCount = ref(0)
let controller: AbortController | null = null

const { data, status, error } = await useKunFetch<KunNewsFeed>('/news', {
  method: 'GET',
  query: { limit: PAGE_SIZE }
})

watch(
  data,
  (page) => {
    if (!page) return
    items.value = page.items
    sources.value = { ...page.sources }
    cursor.value = page.next_cursor
    hasMore.value = !!page.next_cursor
  },
  { immediate: true }
)

const loadMore = async (auto = false) => {
  if (isLoadingMore.value || !hasMore.value || !cursor.value) return
  if (auto) {
    if (autoLoadCount.value >= MAX_AUTO_LOADS) return
    autoLoadCount.value++
  } else {
    autoLoadCount.value = 0
  }
  isLoadingMore.value = true
  controller = new AbortController()
  const next = await kunFetch<KunNewsFeed>('/news', {
    method: 'GET',
    query: { limit: PAGE_SIZE, cursor: cursor.value },
    signal: controller.signal
  })
  isLoadingMore.value = false
  if (!next) return
  items.value.push(...next.items)
  sources.value = { ...sources.value, ...next.sources }
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

// One partner republishes a whole week of bulletins under a single timestamp,
// so a flat list repeats the same date and the same attribution on every card.
// Grouping on (date, source) collapses that into one header per issue — and
// keeps the header truthful, since a group can never span two partners.
const groups = computed<NewsGroup[]>(() => {
  const out: NewsGroup[] = []
  for (const item of items.value) {
    const date = formatDate(item.published_at, { isShowYear: true })
    const key = `${date}-${item.source_key}`
    const last = out.at(-1)
    if (last?.key === key) {
      last.items.push(item)
      continue
    }
    out.push({
      key,
      date,
      source: sources.value[item.source_key],
      items: [item]
    })
  }
  return out
})
</script>

<template>
  <KunLoadingDim
    class="min-w-0"
    :loading="status === 'pending' && items.length > 0"
  >
    <KunNull v-if="error" description="情报服务暂时不可用，请稍后再试" />
    <KunNull
      v-else-if="status !== 'pending' && !items.length"
      description="暂无情报"
    />

    <div v-else class="space-y-8">
      <section v-for="group in groups" :key="group.key" class="space-y-3">
        <div class="space-y-1">
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <KunIcon
              name="lucide:newspaper"
              class="text-primary size-4 shrink-0"
            />
            <span class="text-default-700 font-medium">{{ group.date }}</span>
            <span class="text-default-400 text-xs">
              {{ formatTimeDifference(group.items[0]!.published_at) }}
            </span>
            <KunUserChip
              v-if="group.source?.publisher"
              :user="group.source.publisher"
              size="sm"
              is-navigation
              class-name="ml-auto"
            />
            <KunLink
              v-else-if="group.source"
              :href="group.source.homepage_url"
              target="_blank"
              color="default"
              size="sm"
              underline="hover"
              class-name="ml-auto"
            >
              {{ group.source.name }}
            </KunLink>
          </div>
          <p v-if="group.source?.attribution" class="text-default-400 text-xs">
            {{ group.source.attribution }}
          </p>
        </div>

        <div class="space-y-3">
          <HomeNewsCard
            v-for="item in group.items"
            :key="item.id"
            :item="item"
            :source="group.source"
          />
        </div>
      </section>

      <div v-if="isLoadingMore" class="space-y-3">
        <KunSkeleton
          v-for="n in 3"
          :key="`skeleton-${n}`"
          height="8rem"
          rounded="lg"
        />
      </div>
    </div>

    <div v-if="items.length" ref="sentinel" class="flex justify-center pt-6">
      <KunButton
        v-if="hasMore && !isLoadingMore"
        variant="light"
        @click="loadMore(false)"
      >
        加载更多
      </KunButton>
      <span v-else-if="!hasMore" class="text-default-400 text-sm">
        没有更多情报了
      </span>
    </div>
  </KunLoadingDim>
</template>
