<script setup lang="ts">
// One schema-driven field control (used by EditkitSchemaForm, also standalone
// in the review workbench's amend flow). Renders by the resolved control,
// reflects the projection's capabilities (locked / no-propose → readonly with
// a reason chip), and emits TYPED values upward. Extraction-ready boundary:
// KunUI primitives + self-contained types only.
import { computed, ref, watch } from 'vue'
import type { EditFieldConfig, EditSchemaField } from './types'
import { formatEditItem, formatEditValue, resolveControl } from './utils'

const props = defineProps<{
  field: EditSchemaField
  config?: EditFieldConfig
  modelValue: unknown
  /** Force readonly regardless of capabilities (e.g. while submitting). */
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown]
}>()

const control = computed(() => resolveControl(props.field, props.config))
const label = computed(() => props.config?.label ?? props.field.key)

// Image kinds are display-only this wave (upload pipelines are a later
// wave); deprecated fields render but never edit.
const readonlyReason = computed(() => {
  if (props.field.locked) {
    return '锁定字段'
  }
  if (props.field.deprecated) {
    return '已废弃'
  }
  if (!props.field.can_propose) {
    return '无编辑权限'
  }
  if (
    control.value === 'image' ||
    control.value === 'image-list' ||
    control.value === 'readonly'
  ) {
    return '本期只读'
  }
  return ''
})
const editable = computed(() => !props.disabled && readonlyReason.value === '')

// ---- typed buffers ---------------------------------------------------------
// String-ish controls edit a text buffer and convert on emit; list controls
// edit structured local copies. Buffers re-sync when the upstream value
// changes identity (e.g. the form resets).

const textBuffer = ref('')
const boolBuffer = ref(false)
const stringList = ref<string[]>([])
interface LinkRow {
  name: string
  link: string
  [key: string]: unknown
}
const linkRows = ref<LinkRow[]>([])

const syncFromValue = () => {
  const v = props.modelValue
  switch (control.value) {
    case 'switch':
      boolBuffer.value = v === true
      break
    case 'string-list':
      stringList.value = Array.isArray(v) ? v.map((x) => String(x)) : []
      break
    case 'number-list':
      stringList.value = Array.isArray(v) ? v.map((x) => String(x)) : []
      break
    case 'link-list':
      linkRows.value = Array.isArray(v)
        ? (v as LinkRow[]).map((row) => ({ ...row }))
        : []
      break
    default:
      textBuffer.value = v === null || v === undefined ? '' : String(v)
  }
}
watch(() => props.modelValue, syncFromValue, { immediate: true })

const emitText = (raw: string | number) => {
  const value = String(raw)
  textBuffer.value = value
  switch (control.value) {
    case 'number': {
      const trimmed = value.trim()
      if (trimmed === '') {
        emit('update:modelValue', props.config?.nullable ? null : 0)
        return
      }
      const n = Number(trimmed)
      emit('update:modelValue', Number.isFinite(n) ? n : trimmed)
      return
    }
    case 'date':
      emit('update:modelValue', value === '' ? null : value)
      return
    default:
      emit('update:modelValue', value)
  }
}

const emitSelect = (value: string | number | (string | number)[] | null) => {
  // Single-select only in this form; unwrap a defensive array shape.
  emit('update:modelValue', Array.isArray(value) ? (value[0] ?? null) : value)
}

const emitSwitch = (value: boolean) => {
  boolBuffer.value = value
  emit('update:modelValue', value)
}

const emitStringList = (items: string[]) => {
  stringList.value = items
  if (control.value === 'number-list') {
    emit(
      'update:modelValue',
      items
        .map((x) => Number(x.trim()))
        .filter((n) => Number.isInteger(n) && n > 0)
    )
    return
  }
  emit(
    'update:modelValue',
    items.map((x) => x.trim()).filter((x) => x.length > 0)
  )
}

const emitLinkRows = () => {
  emit(
    'update:modelValue',
    linkRows.value
      .filter((row) => row.name.trim() !== '' || row.link.trim() !== '')
      .map((row) => ({
        ...row,
        name: row.name.trim(),
        link: row.link.trim()
      }))
  )
}

const addLinkRow = () => {
  linkRows.value.push({ name: '', link: '', source: '', source_key: '' })
}

const removeLinkRow = (index: number) => {
  linkRows.value.splice(index, 1)
  emitLinkRows()
}

const selectOptions = computed(() =>
  (props.config?.options ?? []).map((o) => ({ value: o.value, label: o.label }))
)

// Readonly image rendering: single hash/URL or an item list.
const imageURLs = computed(() => {
  const resolve = props.config?.resolveImage
  const toURL = (v: unknown) => (resolve ? resolve(v) : '')
  if (control.value === 'image') {
    const url = toURL(props.modelValue)
    return url ? [url] : []
  }
  if (control.value === 'image-list' && Array.isArray(props.modelValue)) {
    return props.modelValue.map(toURL).filter((u) => u !== '')
  }
  return []
})
</script>

<template>
  <div class="space-y-1">
    <div class="flex items-center gap-2">
      <span class="text-default-700 text-sm font-medium">{{ label }}</span>
      <KunChip v-if="readonlyReason" size="sm" variant="flat" color="default">
        {{ readonlyReason }}
      </KunChip>
    </div>

    <!-- Readonly rendering: images as previews, everything else as text -->
    <template v-if="!editable">
      <div
        v-if="imageURLs.length"
        class="flex flex-wrap items-start gap-2"
      >
        <img
          v-for="(url, i) in imageURLs"
          :key="i"
          :src="url"
          loading="lazy"
          class="max-h-24 max-w-full rounded object-cover"
        />
      </div>
      <p v-else class="text-default-500 text-sm break-all whitespace-pre-wrap">
        {{ formatEditValue(modelValue, config) }}
      </p>
    </template>

    <template v-else>
      <KunTextarea
        v-if="control === 'textarea'"
        :model-value="textBuffer"
        :placeholder="config?.placeholder"
        :description="config?.description"
        @update:model-value="emitText"
      />
      <KunSelect
        v-else-if="control === 'select'"
        :model-value="(modelValue as string | number | null)"
        :options="selectOptions"
        :description="config?.description"
        @update:model-value="emitSelect"
      />
      <KunSwitch
        v-else-if="control === 'switch'"
        :model-value="boolBuffer"
        :label="config?.description ?? ''"
        @update:model-value="emitSwitch"
      />
      <KunTagInput
        v-else-if="control === 'string-list' || control === 'number-list'"
        :model-value="stringList"
        :placeholder="config?.placeholder ?? '输入后回车添加'"
        :description="config?.description"
        @update:model-value="emitStringList"
      />
      <div v-else-if="control === 'link-list'" class="space-y-2">
        <div
          v-for="(row, index) in linkRows"
          :key="index"
          class="flex items-start gap-2"
        >
          <KunInput
            v-model="row.name"
            placeholder="名称"
            class-name="w-1/3"
            @update:model-value="emitLinkRows"
          />
          <KunInput
            v-model="row.link"
            placeholder="https://…"
            class-name="flex-1"
            @update:model-value="emitLinkRows"
          />
          <KunButton
            :is-icon-only="true"
            variant="light"
            color="danger"
            size="sm"
            @click="removeLinkRow(index)"
          >
            <KunIcon name="lucide:x" />
          </KunButton>
        </div>
        <KunButton variant="flat" color="default" size="sm" @click="addLinkRow">
          <KunIcon name="lucide:plus" />
          添加链接
        </KunButton>
        <p v-if="config?.description" class="text-default-400 text-xs">
          {{ config.description }}
        </p>
      </div>
      <KunInput
        v-else
        :model-value="textBuffer"
        :type="control === 'number' ? 'number' : control === 'date' ? 'date' : 'text'"
        :placeholder="config?.placeholder"
        :description="config?.description"
        @update:model-value="emitText"
      />
    </template>
  </div>
</template>
