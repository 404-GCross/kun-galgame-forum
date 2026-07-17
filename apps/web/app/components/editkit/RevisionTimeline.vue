<script setup lang="ts">
// The append-only revision log, newest-first: action badges (with honest
// migrated-history provenance — the legacy_action badge), precise
// changed-field chips, double attribution, and a pick-any-two diff selector.
import { computed, ref, watch } from 'vue'
import type { EditRevision, EditUser } from './types'
import { revisionActionBadge } from './utils'

const props = defineProps<{
  items: EditRevision[]
  users?: Record<number, EditUser>
  labelFor: (key: string) => string
  /** Extra/override labels for legacy action words (host vocabulary). */
  legacyActionLabels?: Record<string, string>
}>()

const emit = defineEmits<{
  diff: [fromSeq: number, toSeq: number]
}>()

const selected = ref<number[]>([])
watch(
  () => props.items,
  () => {
    selected.value = []
  }
)

const toggle = (seq: number) => {
  const index = selected.value.indexOf(seq)
  if (index >= 0) {
    selected.value.splice(index, 1)
    return
  }
  // Keep at most two picks: the oldest pick rolls off.
  if (selected.value.length === 2) {
    selected.value.shift()
  }
  selected.value.push(seq)
}

const canDiff = computed(() => selected.value.length === 2)
const requestDiff = () => {
  if (!canDiff.value) {
    return
  }
  const [a, b] = [...selected.value].sort((x, y) => x - y)
  emit('diff', a!, b!)
}

const legacyLabel = (word: string) =>
  props.legacyActionLabels?.[word] ?? word

const userName = (uid?: number) => {
  if (uid === undefined) {
    return ''
  }
  return props.users?.[uid]?.name ?? `用户 #${uid}`
}
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <p class="text-default-500 text-sm">勾选任意两个版本进行对比</p>
      <KunButton
        size="sm"
        color="primary"
        variant="flat"
        :disabled="!canDiff"
        @click="requestDiff"
      >
        对比所选版本
      </KunButton>
    </div>

    <KunNull v-if="!items.length" description="暂无修订记录" />

    <div v-else class="space-y-2">
      <div
        v-for="revision in items"
        :key="revision.id"
        class="border-default-200 flex items-start gap-3 rounded border p-3"
      >
        <KunCheckBox
          :model-value="selected.includes(revision.seq)"
          @update:model-value="toggle(revision.seq)"
        />
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-default-700 text-sm font-semibold">
              #{{ revision.seq }}
            </span>
            <KunChip
              size="sm"
              variant="flat"
              :color="revisionActionBadge(revision.action).color"
            >
              {{ revisionActionBadge(revision.action).label }}
            </KunChip>
            <KunChip
              v-if="revision.legacy_action"
              size="sm"
              variant="flat"
              color="warning"
            >
              迁移 · {{ legacyLabel(revision.legacy_action) }}
            </KunChip>
            <KunChip
              v-if="revision.legacy_minor"
              size="sm"
              variant="flat"
              color="default"
            >
              小修改
            </KunChip>
            <span class="text-default-400 ml-auto text-xs">
              <KunTime :time="revision.created_at" type="date" show-year />
            </span>
          </div>

          <div v-if="revision.changed_fields?.length" class="flex flex-wrap gap-1">
            <KunChip
              v-for="key in revision.changed_fields"
              :key="key"
              size="sm"
              variant="flat"
              color="default"
            >
              {{ labelFor(key) }}
            </KunChip>
          </div>

          <p v-if="revision.legacy_note" class="text-default-500 text-sm">
            {{ revision.legacy_note }}
          </p>

          <p class="text-default-400 text-xs">
            {{ userName(revision.actor_uid) }}
            <template v-if="revision.amender_uid">
              · 审核修正：{{ userName(revision.amender_uid) }}
            </template>
          </p>

          <!-- Host-supplied per-revision actions (e.g. revert) -->
          <slot name="actions" :revision="revision" />
        </div>
      </div>
    </div>
  </div>
</template>
