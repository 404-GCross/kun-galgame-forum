<script setup lang="ts">
import {
  usePersistQuizGalgameStore,
  type RecentQuizGalgame
} from '~/store/modules/edit/quizGalgame'

// Multi-galgame picker for the 出题 modal. Built on KunAutocomplete (2.11+ async
// @search/:loading/:debounce, 2.12 generic options + #option slot). v-model is
// the selected galgame id array; picks accumulate as removable chips.
type AutoOption = {
  value: string
  label: string
  banner?: string
  thumbhash?: string
  officials?: string[]
}

const props = defineProps<{
  modelValue: number[]
  // Pre-seed the selected chips (edit mode).
  initialSelected?: RecentQuizGalgame[]
}>()
const emits = defineEmits<{ 'update:modelValue': [value: number[]] }>()

const store = usePersistQuizGalgameStore()
const { recent } = storeToRefs(store)

const selectedList = ref<RecentQuizGalgame[]>([])
const query = ref('')
const options = ref<AutoOption[]>([])
const isLoading = ref(false)

const emitIds = () =>
  emits(
    'update:modelValue',
    selectedList.value.map((g) => g.id)
  )

const onSearch = async (raw: string) => {
  const kw = raw.trim()
  if (!kw) {
    options.value = []
    isLoading.value = false
    return
  }
  isLoading.value = true
  const data = await kunFetch<QuizGalgameOption[]>(
    '/galgame-quiz/galgame-search',
    { method: 'GET', query: { keywords: kw } }
  )
  options.value = (data ?? []).map((o) => ({
    value: String(o.id),
    label: getPreferredLanguageText(o.name) || `#${o.id}`,
    banner: o.banner,
    thumbhash: o.banner_thumbhash,
    officials: o.officials
  }))
  isLoading.value = false
}

const pick = (game: RecentQuizGalgame) => {
  if (selectedList.value.some((g) => g.id === game.id)) return
  selectedList.value.push(game)
  emitIds()
  store.add(game) // LRU: bump to front
  query.value = ''
  options.value = []
}

const onSelect = (opt: AutoOption) =>
  pick({
    id: Number(opt.value),
    name: opt.label,
    banner: opt.banner,
    thumbhash: opt.thumbhash,
    officials: opt.officials
  })

const remove = (id: number) => {
  selectedList.value = selectedList.value.filter((g) => g.id !== id)
  emitIds()
}

const officialsText = (g: RecentQuizGalgame) =>
  g.officials?.length ? g.officials.join('、') : ''

// Parent resets modelValue to [] after publishing → drop the local selection.
watch(
  () => props.modelValue,
  (v) => {
    if (!v || v.length === 0) selectedList.value = []
  }
)
// Edit mode: seed the selected chips from the pre-linked games.
watch(
  () => props.initialSelected,
  (v) => {
    if (v && v.length) selectedList.value = v.map((g) => ({ ...g }))
  },
  { immediate: true }
)
</script>

<template>
  <div class="space-y-2">
    <label class="text-sm font-medium">关联 Galgame（可选, 可多选）</label>

    <!-- current selections -->
    <div v-if="selectedList.length" class="space-y-2">
      <div
        v-for="g in selectedList"
        :key="g.id"
        class="border-default-200 flex items-center gap-3 rounded-lg border p-2"
      >
        <div class="bg-default-100 h-12 w-20 shrink-0 overflow-hidden rounded">
          <KunImage
            v-if="g.banner"
            :src="g.banner"
            :thumbhash="g.thumbhash"
            width="80"
            height="48"
            object-fit="cover"
            class-name="h-full w-full"
          />
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{{ g.name }}</p>
          <p v-if="officialsText(g)" class="text-default-500 truncate text-xs">
            {{ officialsText(g) }}
          </p>
        </div>
        <KunButton
          :is-icon-only="true"
          variant="light"
          size="sm"
          @click="remove(g.id)"
        >
          <KunIcon name="lucide:x" />
        </KunButton>
      </div>
    </div>

    <!-- search + recent (always available, so more games can be added) -->
    <KunAutocomplete
      v-model="query"
      :options="options"
      :loading="isLoading"
      :debounce="300"
      manual-filter
      clearable
      placeholder="输入游戏名搜索并关联"
      loading-text="搜索中…"
      no-result-text="无匹配结果"
      @search="onSearch"
      @select="onSelect"
    >
      <template #option="{ option }">
        <div class="flex items-center gap-3">
          <div class="bg-default-100 h-10 w-16 shrink-0 overflow-hidden rounded">
            <KunImage
              v-if="option.banner"
              :src="option.banner"
              :thumbhash="option.thumbhash"
              width="64"
              height="40"
              object-fit="cover"
              class-name="h-full w-full"
            />
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium">{{ option.label }}</p>
            <p
              v-if="option.officials?.length"
              class="text-default-500 truncate text-xs"
            >
              {{ option.officials.join('、') }}
            </p>
          </div>
        </div>
      </template>
    </KunAutocomplete>

    <div v-if="recent.length" class="space-y-1">
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
