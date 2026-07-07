<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import {
  usePersistQuizGalgameStore,
  type RecentQuizGalgame
} from '~/store/modules/edit/quizGalgame'

// Single-galgame picker for the 出题 modal: type-ahead search + a recently-used
// (LRU) quick-pick row. v-model is the selected galgame id (null = unlinked).
const props = defineProps<{ modelValue: number | null }>()
const emits = defineEmits<{ 'update:modelValue': [value: number | null] }>()

const store = usePersistQuizGalgameStore()
const { recent } = storeToRefs(store)

const selected = ref<RecentQuizGalgame | null>(null)
const searchTerm = ref('')
const results = ref<RecentQuizGalgame[]>([])
const isLoading = ref(false)
const isOpen = ref(false)

const doSearch = useDebounceFn(async () => {
  const kw = searchTerm.value.trim()
  if (!kw) {
    results.value = []
    isLoading.value = false
    return
  }
  // /galgame-series/search is the general wiki galgame name search despite its
  // path (the series picker uses the same endpoint) — it returns any matching
  // galgame row, so we reuse it here.
  const data = await kunFetch<GalgameSeriesSearchItem[]>(
    '/galgame-series/search',
    { method: 'GET', query: { keywords: kw } }
  )
  results.value = (data ?? []).map((g) => ({
    id: g.id,
    name: galgameNameFromWire(g, `#${g.id}`)
  }))
  isLoading.value = false
}, 300)

watch(searchTerm, () => {
  isLoading.value = !!searchTerm.value.trim()
  doSearch()
})

const pick = (game: RecentQuizGalgame) => {
  selected.value = game
  emits('update:modelValue', game.id)
  store.add(game) // LRU: bump to front
  searchTerm.value = ''
  results.value = []
  isOpen.value = false
}

const clear = () => {
  selected.value = null
  emits('update:modelValue', null)
}

const onBlur = () => setTimeout(() => (isOpen.value = false), 150)

// Parent resets modelValue to null after publishing → drop the local selection.
watch(
  () => props.modelValue,
  (v) => {
    if (v === null) selected.value = null
  }
)
</script>

<template>
  <div class="space-y-2">
    <label class="text-sm font-medium">关联 Galgame（可选）</label>

    <!-- current selection -->
    <div v-if="selected" class="flex items-center gap-2">
      <KunChip color="primary" variant="flat">{{ selected.name }}</KunChip>
      <KunButton :is-icon-only="true" variant="light" size="sm" @click="clear">
        <KunIcon name="lucide:x" />
      </KunButton>
    </div>

    <!-- search box + results dropdown -->
    <div v-else class="relative">
      <KunInput
        v-model="searchTerm"
        placeholder="输入游戏名搜索并关联（可选）"
        is-clearable
        @focus="isOpen = true"
        @blur="onBlur"
      />
      <div
        v-if="isOpen && searchTerm.trim()"
        class="border-default-200 bg-background absolute z-20 mt-1 max-h-60 w-full overflow-auto rounded-lg border shadow-lg"
      >
        <p v-if="isLoading" class="text-default-500 px-3 py-2 text-sm">
          搜索中...
        </p>
        <button
          v-for="g in results"
          :key="g.id"
          type="button"
          class="hover:bg-default-100 block w-full truncate px-3 py-2 text-left text-sm"
          @mousedown.prevent="pick(g)"
        >
          {{ g.name }}
        </button>
        <p
          v-if="!isLoading && !results.length"
          class="text-default-500 px-3 py-2 text-sm"
        >
          无匹配结果
        </p>
      </div>
    </div>

    <!-- recently associated (LRU) quick picks -->
    <div
      v-if="!selected && recent.length"
      class="flex flex-wrap items-center gap-2"
    >
      <span class="text-default-400 text-xs">最近关联</span>
      <KunButton
        v-for="g in recent"
        :key="g.id"
        variant="flat"
        size="sm"
        @click="pick(g)"
      >
        {{ g.name }}
      </KunButton>
    </div>
  </div>
</template>
