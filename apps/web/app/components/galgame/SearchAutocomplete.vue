<script setup lang="ts">
type GalgameSearchHit = {
  id: number
  name: string
  banner?: string
  thumbhash?: string
  officials?: string[]
}

type AutoOption = GalgameSearchHit & { value: string; label: string }

const props = defineProps<{
  excludeIds?: number[]
  placeholder?: string
}>()
const emits = defineEmits<{ select: [hit: GalgameSearchHit] }>()

const query = ref('')
const options = ref<AutoOption[]>([])
const isLoading = ref(false)

let searchSeq = 0

const onSearch = async (raw: string) => {
  const kw = raw.trim()
  const seq = ++searchSeq
  if (!kw) {
    options.value = []
    isLoading.value = false
    return
  }
  isLoading.value = true
  const data = await kunFetch<QuizGalgameOption[]>('/galgame/search/picker', {
    method: 'GET',
    query: { keywords: kw }
  })
  if (seq !== searchSeq) return
  const exclude = new Set(props.excludeIds ?? [])
  options.value = (data ?? [])
    .filter((o) => !exclude.has(o.id))
    .map((o) => {
      const name = getPreferredLanguageText(o.name) || `#${o.id}`
      return {
        id: o.id,
        name,
        banner: o.banner,
        thumbhash: o.banner_thumbhash,
        officials: o.officials,
        value: String(o.id),
        label: name
      }
    })
  isLoading.value = false
}

const onSelect = (opt: AutoOption) => {
  emits('select', {
    id: opt.id,
    name: opt.name,
    banner: opt.banner,
    thumbhash: opt.thumbhash,
    officials: opt.officials
  })
  query.value = ''
  options.value = []
}
</script>

<template>
  <KunAutocomplete
    v-model="query"
    :options="options"
    :loading="isLoading"
    :debounce="300"
    manual-filter
    clearable
    :placeholder="placeholder ?? '输入游戏名搜索'"
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
</template>
