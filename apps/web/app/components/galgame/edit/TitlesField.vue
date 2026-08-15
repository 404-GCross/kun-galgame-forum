<script setup lang="ts">
interface TitleRow {
  lang: string
  title: string
  kind: number
  latin?: string
}

const props = defineProps<{
  modelValue: unknown
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown]
}>()

const KIND_OFFICIAL = 0
const KIND_ALIAS = 1
const KIND_ABBREVIATION = 2

const KIND_LABEL: Record<number, string> = {
  [KIND_OFFICIAL]: '官方名',
  [KIND_ALIAS]: '别名',
  [KIND_ABBREVIATION]: '缩写'
}

const LANG_OPTIONS = [
  { value: 'ja', label: '日语' },
  { value: 'zh-Hans', label: '简中' },
  { value: 'zh-Hant', label: '繁中' },
  { value: 'en', label: '英语' }
]

// Only an alias may carry the empty language; catalog rejects an official or an
// abbreviation without one.
const ALIAS_LANG_OPTIONS = [{ value: '', label: '无语言' }, ...LANG_OPTIONS]

const KIND_OPTIONS = [
  { value: String(KIND_ALIAS), label: '别名' },
  { value: String(KIND_ABBREVIATION), label: '缩写' }
]

// A freshly added row has an empty title, which the value we hand upwards must
// not contain — catalog rejects it. Keeping a local copy lets the blank row
// live in the form until it is typed into.
const local = ref<TitleRow[]>([])
let lastEmitted = ''

watch(
  () => props.modelValue,
  (value) => {
    const incoming = Array.isArray(value) ? (value as TitleRow[]) : []
    if (JSON.stringify(incoming) === lastEmitted) {
      return
    }
    local.value = incoming.map((row) => ({ ...row }))
  },
  { immediate: true, deep: true }
)

const push = () => {
  const cleaned = local.value.filter((row) => row.title.trim() !== '')
  lastEmitted = JSON.stringify(cleaned)
  emit('update:modelValue', cleaned)
}

const indexed = computed(() =>
  local.value.map((row, index) => ({ row, index }))
)
const officials = computed(() =>
  indexed.value.filter((r) => r.row.kind === KIND_OFFICIAL)
)
const others = computed(() =>
  indexed.value.filter((r) => r.row.kind !== KIND_OFFICIAL)
)

const patchRow = (index: number, patch: Partial<TitleRow>) => {
  const row = local.value[index]
  if (!row) {
    return
  }
  local.value[index] = { ...row, ...patch }
  push()
}

const removeRow = (index: number) => {
  local.value.splice(index, 1)
  push()
}

const addOfficial = () => {
  local.value.push({ lang: 'ja', title: '', kind: KIND_OFFICIAL })
}

const draft = ref('')
const addAlias = () => {
  const title = draft.value.trim()
  if (!title) {
    return
  }
  draft.value = ''
  local.value.push({ lang: '', title, kind: KIND_ALIAS })
  push()
}

const detailed = ref(false)
const PREVIEW = 10
const expanded = ref(false)
const visibleOthers = computed(() =>
  expanded.value || detailed.value
    ? others.value
    : others.value.slice(0, PREVIEW)
)
const foldedCount = computed(
  () => others.value.length - visibleOthers.value.length
)

const chipTitle = (row: TitleRow) => {
  const lang = ALIAS_LANG_OPTIONS.find((o) => o.value === row.lang)?.label
  return `${lang ?? row.lang} · ${KIND_LABEL[row.kind] ?? ''}`
}
</script>

<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <p class="text-default-500 text-xs font-medium">
        官方名 · 每种语言一条，条目名从这里按原始语言挑选
      </p>
      <div
        v-for="entry in officials"
        :key="entry.index"
        class="flex items-start gap-2"
      >
        <KunSelect
          :model-value="entry.row.lang"
          :options="LANG_OPTIONS"
          class-name="w-28 shrink-0"
          :disabled="disabled"
          @update:model-value="
            (v: string | string[] | null) =>
              patchRow(entry.index, { lang: String(v ?? '') })
          "
        />
        <KunInput
          :model-value="entry.row.title"
          placeholder="官方标题"
          class-name="flex-1"
          :disabled="disabled"
          @update:model-value="
            (v: string | number) => patchRow(entry.index, { title: String(v) })
          "
        />
        <KunButton
          :is-icon-only="true"
          variant="light"
          color="danger"
          size="sm"
          :disabled="disabled || officials.length <= 1"
          title="删除"
          @click="removeRow(entry.index)"
        >
          <KunIcon name="lucide:x" />
        </KunButton>
      </div>
      <KunButton
        variant="flat"
        color="default"
        size="sm"
        :disabled="disabled"
        @click="addOfficial"
      >
        <KunIcon name="lucide:plus" />
        添加官方名
      </KunButton>
    </div>

    <div class="space-y-2">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <p class="text-default-500 text-xs font-medium">
          别名与缩写 · 玩家常用的其他叫法（{{ others.length }}）
        </p>
        <KunButton
          v-if="others.length"
          variant="light"
          color="default"
          size="sm"
          @click="detailed = !detailed"
        >
          <KunIcon name="lucide:settings-2" />
          {{ detailed ? '收起' : '逐条编辑' }}
        </KunButton>
      </div>

      <template v-if="detailed">
        <div
          v-for="entry in visibleOthers"
          :key="entry.index"
          class="flex items-start gap-2"
        >
          <KunSelect
            :model-value="entry.row.lang"
            :options="ALIAS_LANG_OPTIONS"
            class-name="w-28 shrink-0"
            :disabled="disabled"
            @update:model-value="
              (v: string | string[] | null) =>
                patchRow(entry.index, { lang: String(v ?? '') })
            "
          />
          <KunInput
            :model-value="entry.row.title"
            placeholder="别名"
            class-name="flex-1"
            :disabled="disabled"
            @update:model-value="
              (v: string | number) =>
                patchRow(entry.index, { title: String(v) })
            "
          />
          <KunSelect
            :model-value="String(entry.row.kind)"
            :options="KIND_OPTIONS"
            class-name="w-24 shrink-0"
            :disabled="disabled"
            @update:model-value="
              (v: string | string[] | null) =>
                patchRow(entry.index, { kind: Number(v ?? KIND_ALIAS) })
            "
          />
          <KunButton
            :is-icon-only="true"
            variant="light"
            color="danger"
            size="sm"
            :disabled="disabled"
            title="删除"
            @click="removeRow(entry.index)"
          >
            <KunIcon name="lucide:x" />
          </KunButton>
        </div>
      </template>

      <div v-else-if="others.length" class="flex flex-wrap gap-1.5">
        <KunChip
          v-for="entry in visibleOthers"
          :key="entry.index"
          size="sm"
          variant="flat"
          color="default"
          :closable="!disabled"
          :title="chipTitle(entry.row)"
          @close="removeRow(entry.index)"
        >
          {{ entry.row.title }}
          <span
            v-if="entry.row.kind === KIND_ABBREVIATION"
            class="text-default-400 text-xs"
          >
            缩写
          </span>
        </KunChip>
      </div>

      <KunButton
        v-if="foldedCount > 0"
        variant="light"
        color="default"
        size="sm"
        @click="expanded = true"
      >
        还有 {{ foldedCount }} 条，全部显示
      </KunButton>

      <div v-if="!disabled" class="flex items-start gap-2">
        <KunInput
          v-model="draft"
          placeholder="输入别名后回车添加"
          class-name="flex-1"
          @keydown.enter.prevent="addAlias"
        />
        <KunButton
          variant="flat"
          color="default"
          size="sm"
          :disabled="!draft.trim()"
          @click="addAlias"
        >
          <KunIcon name="lucide:plus" />
          添加
        </KunButton>
      </div>
    </div>
  </div>
</template>
