<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import {
  usePersistQuizGalgameStore,
  type RecentQuizGalgame
} from '~/store/modules/edit/quizGalgame'

// Single-galgame picker for the 出题 modal: type-ahead search (banner + 会社
// enriched) + a recently-used (LRU) quick-pick row. v-model is the selected
// galgame id (null = unlinked).
const props = defineProps<{ modelValue: number | null }>()
const emits = defineEmits<{ 'update:modelValue': [value: number | null] }>()

const store = usePersistQuizGalgameStore()
const { recent } = storeToRefs(store)

const selected = ref<RecentQuizGalgame | null>(null)
const searchTerm = ref('')
const results = ref<RecentQuizGalgame[]>([])
const isLoading = ref(false)
const isOpen = ref(false)

const toRecent = (o: QuizGalgameOption): RecentQuizGalgame => ({
  id: o.id,
  name: getPreferredLanguageText(o.name) || `#${o.id}`,
  banner: o.banner,
  thumbhash: o.banner_thumbhash,
  officials: o.officials
})

const doSearch = useDebounceFn(async () => {
  const kw = searchTerm.value.trim()
  if (!kw) {
    results.value = []
    isLoading.value = false
    return
  }
  const data = await kunFetch<QuizGalgameOption[]>(
    '/galgame-quiz/galgame-search',
    { method: 'GET', query: { keywords: kw } }
  )
  results.value = (data ?? []).map(toRecent)
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

const officialsText = (g: RecentQuizGalgame) =>
  g.officials?.length ? g.officials.join('、') : ''

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
    <div
      v-if="selected"
      class="border-default-200 flex items-center gap-3 rounded-lg border p-2"
    >
      <div
        class="bg-default-100 h-12 w-20 shrink-0 overflow-hidden rounded"
      >
        <KunImage
          v-if="selected.banner"
          :src="selected.banner"
          :thumbhash="selected.thumbhash"
          width="80"
          height="48"
          object-fit="cover"
          class-name="h-full w-full"
        />
      </div>
      <div class="min-w-0 flex-1">
        <p class="truncate font-medium">{{ selected.name }}</p>
        <p
          v-if="officialsText(selected)"
          class="text-default-500 truncate text-xs"
        >
          {{ officialsText(selected) }}
        </p>
      </div>
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
        class="border-default-200 absolute z-20 mt-1 max-h-72 w-full overflow-auto rounded-lg border bg-[oklch(var(--content1))] shadow-lg"
      >
        <p v-if="isLoading" class="text-default-500 px-3 py-2 text-sm">
          搜索中...
        </p>
        <button
          v-for="g in results"
          :key="g.id"
          type="button"
          class="hover:bg-default-100 flex w-full items-center gap-3 px-3 py-2 text-left"
          @mousedown.prevent="pick(g)"
        >
          <div class="bg-default-100 h-10 w-16 shrink-0 overflow-hidden rounded">
            <KunImage
              v-if="g.banner"
              :src="g.banner"
              :thumbhash="g.thumbhash"
              width="64"
              height="40"
              object-fit="cover"
              class-name="h-full w-full"
            />
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium">{{ g.name }}</p>
            <p
              v-if="officialsText(g)"
              class="text-default-500 truncate text-xs"
            >
              {{ officialsText(g) }}
            </p>
          </div>
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
    <div v-if="!selected && recent.length" class="space-y-1">
      <span class="text-default-400 text-xs">最近关联</span>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="g in recent"
          :key="g.id"
          type="button"
          class="border-default-200 hover:border-primary flex items-center gap-2 rounded-lg border p-1 pr-2 transition-colors"
          @click="pick(g)"
        >
          <div class="bg-default-100 h-8 w-12 shrink-0 overflow-hidden rounded">
            <KunImage
              v-if="g.banner"
              :src="g.banner"
              :thumbhash="g.thumbhash"
              width="48"
              height="32"
              object-fit="cover"
              class-name="h-full w-full"
            />
          </div>
          <span class="max-w-[10rem] truncate text-sm">{{ g.name }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
