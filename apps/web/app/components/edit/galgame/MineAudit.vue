<script setup lang="ts">
const limit = 20

const items = ref<UserClaimItem[]>([])
const nextBefore = ref(0)
const total = ref(0)
const isLoadingMore = ref(false)

const { data, refresh } = await useKunFetch<UserClaimList>('/galgame/myaudit', {
  query: { before: 0, limit }
})

watch(
  data,
  (page) => {
    if (!page) {
      return
    }
    items.value = page.items
    nextBefore.value = page.next_before
    total.value = page.total
  },
  { immediate: true }
)

const hasMore = computed(
  () => nextBefore.value > 0 && items.value.length < total.value
)

const loadMore = async () => {
  if (isLoadingMore.value || !hasMore.value) {
    return
  }
  isLoadingMore.value = true
  const next = await kunFetch<UserClaimList>(
    `/galgame/myaudit?before=${nextBefore.value}&limit=${limit}`
  )
  isLoadingMore.value = false
  if (!next) {
    return
  }
  items.value.push(...next.items)
  nextBefore.value = next.next_before
}

const stateBadge = galgameClaimStateBadge
const nameOf = (item: UserClaimItem) => item.display_name || '(无标题)'
const gidOf = (item: UserClaimItem) => item.product_work_id ?? item.work_id
</script>

<template>
  <div class="space-y-4">
    <KunHeader
      name="我的 Galgame 审核"
      description="您审核过的 Galgame 申请, 包括已通过 / 已拒绝 / 已下架的作品。"
    />

    <KunDivider />

    <KunInfo
      v-if="!data"
      color="danger"
      title="加载失败"
      description="无法获取您的审核列表, 可能是后端 / Galgame 资料库暂时不可用, 请稍后重试。"
    />

    <div v-else-if="items.length" class="flex flex-col gap-3">
      <div
        v-for="item in items"
        :key="item.work_id"
        class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
      >
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <h3
              class="hover:text-primary truncate text-lg font-medium transition-colors"
            >
              {{ nameOf(item) }}
            </h3>
            <KunChip
              size="xs"
              variant="flat"
              :color="stateBadge(item.claim_state).color"
            >
              {{ stateBadge(item.claim_state).label }}
            </KunChip>
          </div>
          <div
            class="text-default-500 flex flex-wrap items-center gap-2 text-sm"
          >
            <span>首次审核 <KunTime :time="item.first_acted_at" /></span>
            <template v-if="item.last_event_at !== item.first_acted_at">
              <span>·</span>
              <span>最后处理 <KunTime :time="item.last_event_at" /></span>
            </template>
          </div>
          <div
            v-if="item.last_reason"
            class="text-default-500 bg-default-500/10 mt-1 rounded-md px-2 py-1 text-sm"
          >
            审核理由: {{ item.last_reason }}
          </div>
        </div>
        <div class="flex shrink-0 gap-2">
          <KunLink :to="`/galgame/${gidOf(item)}`">
            <KunButton size="sm" variant="flat">查看</KunButton>
          </KunLink>
        </div>
      </div>
    </div>

    <KunNull v-else-if="data && !items.length" />

    <KunButton
      v-if="hasMore"
      variant="flat"
      :loading="isLoadingMore"
      @click="loadMore"
    >
      加载更多
    </KunButton>
  </div>
</template>
