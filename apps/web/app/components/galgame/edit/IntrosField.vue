<script setup lang="ts">
interface IntroRow {
  lang: string
  intro: string
}

const props = defineProps<{
  modelValue: unknown
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown]
}>()

// catalog speaks BCP-47-ish tags; the editor speaks the forum's locale keys.
const LANGS = [
  { value: 'zh-Hans', textValue: '简体中文', locale: 'zh-cn' },
  { value: 'zh-Hant', textValue: '繁体中文', locale: 'zh-tw' },
  { value: 'ja', textValue: '日本語', locale: 'ja-jp' },
  { value: 'en', textValue: 'English', locale: 'en-us' }
] as const

const rows = computed<IntroRow[]>(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as IntroRow[]) : []
)

const textOf = (lang: string) =>
  rows.value.find((row) => row.lang === lang)?.intro ?? ''

const active = ref<string>(
  LANGS.find((l) => rows.value.some((row) => row.lang === l.value))?.value ??
    'zh-Hans'
)

const activeLocale = computed<Language>(
  () =>
    (LANGS.find((l) => l.value === active.value)?.locale ?? 'zh-cn') as Language
)

const items = computed(() =>
  LANGS.map((lang) => ({
    value: lang.value,
    textValue: lang.textValue,
    filled: textOf(lang.value).trim().length > 0
  }))
)

const setText = (lang: string, text: string) => {
  const kept = new Map(rows.value.map((row) => [row.lang, row.intro]))
  if (text.trim()) {
    kept.set(lang, text)
  } else {
    kept.delete(lang)
  }
  emit(
    'update:modelValue',
    LANGS.filter((l) => kept.has(l.value)).map((l) => ({
      lang: l.value,
      intro: kept.get(l.value) ?? ''
    }))
  )
}

const clearActive = () => setText(active.value, '')

const filledCount = computed(() => items.value.filter((i) => i.filled).length)
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <KunTab
        :model-value="active"
        :items="items"
        orientation="horizontal"
        variant="underlined"
        color="primary"
        size="sm"
        @update:model-value="(value) => (active = value)"
      >
        <template #tab="{ item }">
          {{ item.textValue }}
          <KunBadge
            v-if="item.filled"
            variant="dot"
            color="success"
            size="sm"
          />
        </template>
      </KunTab>

      <span class="text-default-400 text-xs">
        已填写 {{ filledCount }} / {{ LANGS.length }} 种语言
      </span>
    </div>

    <KunMilkdownDualEditorProvider
      v-if="!disabled"
      :key="active"
      :value-markdown="textOf(active)"
      :language="activeLocale"
      :disable-image="true"
      placeholder="支持 Markdown。可参考官方页面、DLsite 的 ストーリー、Bangumi，也可以自行撰写"
      @set-markdown="(text: string) => setText(active, text)"
    >
      <KunButton
        v-if="textOf(active)"
        variant="light"
        color="danger"
        size="sm"
        @click="clearActive"
      >
        清空该语言
      </KunButton>
    </KunMilkdownDualEditorProvider>

    <p v-else class="text-default-500 text-sm whitespace-pre-wrap">
      {{ textOf(active) || '（空）' }}
    </p>
  </div>
</template>
