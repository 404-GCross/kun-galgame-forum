<script setup lang="ts">
// GALGAME_EDIT — the actor edited a galgame; shows the embedded galgame preview
// plus the wiki revision diff (same renderer as the history page), lazily loaded
// and collapsed to 100px with a 显示更多 toggle (mirrors the resource-note card).
//
// SnapshotDiff mirrors the GET /galgame/:gid/revisions/:rev/diff payload (the
// type isn't exported from useRevisionHistory, so it's redeclared here).
interface SnapshotDiff {
  changed_keys: Record<string, boolean>
  old: Record<string, unknown>
  new: Record<string, unknown>
  names?: {
    tags?: Record<string, string>
    officials?: Record<string, string>
    engines?: Record<string, string>
    series?: Record<string, string>
  }
}

const props = defineProps<{ activity: ActivityItem }>()
const data = computed(
  () => props.activity.data as GalgameActivityData | undefined
)

const diff = ref<SnapshotDiff | null>(null)
const isLoading = ref(false)

const loadDiff = async () => {
  const gid = data.value?.galgame_id
  if (!gid || diff.value || isLoading.value) return
  isLoading.value = true
  try {
    // The diff endpoint keys on the per-galgame revision NUMBER. The activity now
    // carries it directly (revision_number). Legacy rows synced before the wiki
    // feed exposed it have only the revision ROW id, so fall back to resolving
    // id → number via the history list (newest-first — a recent edit is page 1).
    let number = data.value?.revision_number
    if (!number) {
      const rowId = data.value?.revision_id
      if (!rowId) return
      const history = await kunFetch<{
        items?: { id: number; revision: number }[]
      }>(`/galgame/${gid}/history/all?page=1&limit=100`)
      number = history?.items?.find((r) => r.id === rowId)?.revision
    }
    if (!number) return
    const res = await kunFetch<SnapshotDiff>(
      `/galgame/${gid}/revisions/${number}/diff`
    )
    if (res) diff.value = res
  } finally {
    isLoading.value = false
  }
}
onMounted(loadDiff)

// Collapse the diff to 100px with a 显示更多 toggle (resource-note pattern).
const DIFF_COLLAPSED_MAX_HEIGHT = 100
const diffRef = ref<HTMLElement | null>(null)
const isExpanded = ref(false)
const isOverflowing = ref(false)
let resizeObserver: ResizeObserver | null = null

const measureOverflow = () => {
  const el = diffRef.value
  if (!el) {
    isOverflowing.value = false
    return
  }
  isOverflowing.value = el.scrollHeight > DIFF_COLLAPSED_MAX_HEIGHT
}

const diffStyle = computed(() => {
  if (!isOverflowing.value || isExpanded.value) return undefined
  return { maxHeight: `${DIFF_COLLAPSED_MAX_HEIGHT}px`, overflow: 'hidden' }
})

watch(diff, () =>
  nextTick(() => {
    if (diffRef.value && !resizeObserver) {
      resizeObserver = new ResizeObserver(() => measureOverflow())
      resizeObserver.observe(diffRef.value)
    }
    measureOverflow()
  })
)

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
})
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-3">
      <p class="text-default-600 text-sm break-all">
        编辑了《{{ data?.name || activity.content }}》
      </p>

      <ActivityCardGalgameInfo :activity="activity" />

      <div v-if="isLoading" class="text-default-400 text-sm">
        加载编辑内容…
      </div>
      <div
        v-else-if="diff"
        class="border-default-200 rounded-lg border p-2 text-sm"
      >
        <div ref="diffRef" :style="diffStyle">
          <GalgameSnapshotDiff
            :changed-keys="diff.changed_keys"
            :old-snap="diff.old"
            :new-snap="diff.new"
            :names="diff.names"
          />
        </div>
        <button
          v-if="isOverflowing"
          type="button"
          class="text-primary mt-1 flex items-center gap-1 text-sm"
          @click="isExpanded = !isExpanded"
        >
          {{ isExpanded ? '收起' : '显示更多' }}
          <KunIcon
            :name="isExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
            class="size-4"
          />
        </button>
      </div>

      <div class="flex items-center justify-between gap-2 text-sm">
        <span class="text-default-500">该更新已经被合并到 鲲Galgame百科</span>
        <KunLink
          underline="none"
          color="default"
          :to="activity.link"
          class-name="text-default-500 hover:text-primary flex shrink-0 items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
