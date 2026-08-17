<script setup lang="ts">
import { ref, watch } from 'vue'
import type { EditSelectOption } from '~/components/editkit/types'

interface CreditRow {
  role_id: number
  credit_name_id: number
  character_id?: number
  note?: string
}

const ROLE_OPTIONS: EditSelectOption[] = [
  { value: '247', label: '脚本' },
  { value: '184', label: '原画' },
  { value: '145', label: '人设' },
  { value: '209', label: '音乐' },
  { value: '173', label: '导演' },
  { value: '1', label: '声优' },
  { value: '286', label: '演唱' },
  { value: '199', label: '作词' },
  { value: '158', label: '作曲' },
  { value: '115', label: '编曲' },
  { value: '3', label: '翻译' },
  { value: '4', label: '编辑' },
  { value: '5', label: 'QA' },
  { value: '2', label: '其他' }
]

const props = defineProps<{
  modelValue: unknown
  disabled?: boolean
  searchNames: (keyword: string) => Promise<EditSelectOption[]>
  searchCharacters: (keyword: string) => Promise<EditSelectOption[]>
  resolveNames?: (
    ids: (string | number)[]
  ) => Promise<EditSelectOption[]> | EditSelectOption[]
  resolveCharacters?: (
    ids: (string | number)[]
  ) => Promise<EditSelectOption[]> | EditSelectOption[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown]
}>()

const asRows = (value: unknown): CreditRow[] => {
  if (!Array.isArray(value)) {
    return []
  }
  return (value as CreditRow[]).map((row) => {
    const next: CreditRow = {
      role_id: Number(row.role_id),
      credit_name_id: Number(row.credit_name_id)
    }
    if (row.character_id) {
      next.character_id = Number(row.character_id)
    }
    if (row.note) {
      next.note = String(row.note)
    }
    return next
  })
}

const local = ref<CreditRow[]>([])
let lastEmitted = ''

watch(
  () => props.modelValue,
  (value) => {
    const incoming = asRows(value)
    if (JSON.stringify(incoming) === lastEmitted) {
      return
    }
    local.value = incoming
  },
  { immediate: true }
)

const cleaned = (list: CreditRow[]) =>
  list
    .filter((row) => row.credit_name_id > 0 && row.role_id > 0)
    .map((row) => {
      const next: CreditRow = {
        role_id: row.role_id,
        credit_name_id: row.credit_name_id
      }
      if (row.character_id) {
        next.character_id = row.character_id
      }
      if (row.note?.trim()) {
        next.note = row.note.trim()
      }
      return next
    })

const push = () => {
  const next = cleaned(local.value)
  lastEmitted = JSON.stringify(next)
  emit('update:modelValue', next)
}

const patchRow = (index: number, patch: Partial<CreditRow>) => {
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

const addRow = () => {
  local.value.push({ role_id: 247, credit_name_id: 0 })
}

const roleLabel = (id: number) =>
  ROLE_OPTIONS.find((o) => String(o.value) === String(id))?.label ??
  `职务 #${id}`
</script>

<template>
  <div class="space-y-3">
    <div
      v-for="(row, index) in local"
      :key="`${row.role_id}-${row.credit_name_id}-${index}`"
      class="space-y-2 rounded-lg border border-default-200 p-3"
    >
      <div class="flex flex-wrap items-start gap-2">
        <KunSelect
          :model-value="String(row.role_id)"
          :options="ROLE_OPTIONS"
          class-name="w-28 shrink-0"
          :disabled="disabled"
          @update:model-value="
            (v: string | number | (string | number)[] | null) =>
              patchRow(index, { role_id: Number(v ?? 0) })
          "
        />
        <div class="min-w-40 flex-1">
          <EditkitEntityPicker
            :model-value="row.credit_name_id || null"
            :disabled="disabled"
            placeholder="搜索署名名义"
            :search="searchNames"
            :resolve="resolveNames"
            @update:model-value="
              (value) =>
                patchRow(index, { credit_name_id: Number(value ?? 0) })
            "
          />
        </div>
        <KunButton
          :is-icon-only="true"
          variant="light"
          color="danger"
          size="sm"
          :disabled="disabled"
          title="删除"
          @click="removeRow(index)"
        >
          <KunIcon name="lucide:x" />
        </KunButton>
      </div>
      <div class="flex flex-wrap items-start gap-2">
        <div class="min-w-40 flex-1">
          <EditkitEntityPicker
            :model-value="row.character_id || null"
            :disabled="disabled"
            placeholder="配音角色（可选）"
            :search="searchCharacters"
            :resolve="resolveCharacters"
            @update:model-value="
              (value) => {
                const id = Number(value ?? 0)
                patchRow(index, { character_id: id > 0 ? id : undefined })
              }
            "
          />
        </div>
        <KunInput
          :model-value="row.note ?? ''"
          placeholder="备注（可选）"
          class-name="min-w-40 flex-1"
          :disabled="disabled"
          @update:model-value="
            (v: string | number) => patchRow(index, { note: String(v) })
          "
        />
      </div>
      <p class="text-default-400 text-xs">{{ roleLabel(row.role_id) }}</p>
    </div>

    <KunButton
      variant="flat"
      color="default"
      size="sm"
      :disabled="disabled"
      @click="addRow"
    >
      <KunIcon name="lucide:plus" />
      添加本站署名
    </KunButton>
  </div>
</template>
