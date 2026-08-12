<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'

const { searchHistory } = storeToRefs(usePersistKUNGalgameSearchStore())
const { keywords } = storeToRefs(useTempSearchStore())

const isFocus = ref(false)
const input = ref<HTMLElement | null>(null)
const inputValue = ref('')

watchDebounced(
  () => inputValue.value,
  (value) => {
    keywords.value = value.trim()
  },
  { debounce: 500 }
)

watch(
  keywords,
  (value) => {
    if (value !== inputValue.value.trim()) {
      inputValue.value = value
    }
  },
  { immediate: true }
)

onMounted(() => input.value?.focus())

const handleInputBlur = () => {
  isFocus.value = false
  if (!keywords.value.trim()) {
    return
  }

  if (!searchHistory.value.includes(keywords.value)) {
    searchHistory.value.push(keywords.value)
  }
}

const handleEnter = () => {
  keywords.value = inputValue.value.trim()
}
</script>

<template>
  <KunInput
    ref="input"
    v-model="inputValue"
    type="search"
    size="lg"
    :color="isFocus ? 'primary' : 'default'"
    placeholder="输入内容以自动搜索"
    @focus="isFocus = true"
    @blur="handleInputBlur"
    @keydown.enter="handleEnter"
  />
</template>
