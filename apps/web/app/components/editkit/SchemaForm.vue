<script setup lang="ts">
// Schema-driven edit form (infra doc 21 §2.7): renders every projected field
// by its kind + the host's presentation config, tracks a working copy, and
// emits ONLY the dirty subset as the proposal patch (field key → new value).
// Zero policy logic — capabilities come from the projection.
import { computed, reactive, toRaw, watch } from 'vue'
import type {
  EditFieldConfigMap,
  EditSchemaField
} from './types'
import { editValueEqual } from './utils'

const props = defineProps<{
  fields: EditSchemaField[]
  /** Current entity values keyed by eternal field keys. */
  values: Record<string, unknown>
  config: EditFieldConfigMap
  /** Section order; fields whose config.group is absent land in the last
   * unnamed section. */
  groupOrder?: string[]
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:patch': [patch: Record<string, unknown>]
}>()

// Working copy, (re)seeded whenever the upstream values change identity.
const working = reactive<Record<string, unknown>>({})
watch(
  () => props.values,
  (values) => {
    for (const key of Object.keys(working)) {
      Reflect.deleteProperty(working, key)
    }
    for (const field of props.fields) {
      working[field.key] = structuredClone(toRaw(values[field.key]) ?? null)
    }
  },
  { immediate: true, deep: false }
)

const patch = computed<Record<string, unknown>>(() => {
  const out: Record<string, unknown> = {}
  for (const field of props.fields) {
    if (field.locked || field.deprecated || !field.can_propose) {
      continue
    }
    const baseline = props.values[field.key] ?? null
    const current = working[field.key] ?? null
    if (!editValueEqual(baseline, current)) {
      out[field.key] = current
    }
  }
  return out
})
watch(patch, (value) => emit('update:patch', value), { deep: false })

const dirtyCount = computed(() => Object.keys(patch.value).length)
defineExpose({ dirtyCount })

// Group fields into ordered sections.
const sections = computed(() => {
  const byGroup = new Map<string, EditSchemaField[]>()
  for (const field of props.fields) {
    if (field.deprecated) {
      continue
    }
    const group = props.config[field.key]?.group ?? ''
    const bucket = byGroup.get(group)
    if (bucket) {
      bucket.push(field)
    } else {
      byGroup.set(group, [field])
    }
  }
  const order = props.groupOrder ?? [...byGroup.keys()]
  const out: { name: string; fields: EditSchemaField[] }[] = []
  for (const name of order) {
    const fields = byGroup.get(name)
    if (fields?.length) {
      out.push({ name, fields })
      byGroup.delete(name)
    }
  }
  for (const [name, fields] of byGroup) {
    out.push({ name, fields })
  }
  return out
})
</script>

<template>
  <div class="space-y-6">
    <section v-for="section in sections" :key="section.name" class="space-y-3">
      <h3
        v-if="section.name"
        class="text-default-900 border-b pb-1 text-base font-semibold"
      >
        {{ section.name }}
      </h3>
      <div class="grid grid-cols-1 gap-4">
        <EditkitSchemaField
          v-for="field in section.fields"
          :key="field.key"
          v-model="working[field.key]"
          :field="field"
          :config="config[field.key]"
          :disabled="disabled"
        />
      </div>
    </section>
  </div>
</template>
