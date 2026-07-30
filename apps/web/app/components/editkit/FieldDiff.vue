<script setup lang="ts">
// One field's old→new rendering, by the registry's diff hint (doc 21 §2.4):
// text (word-level unified diff), item-level lists, side-by-side images.
//
// Both text regimes — the `lines` hint for wiki intros and the bare scalar
// fallback — go through TextDiff. The `lines` hint used to run a line-level LCS
// and the scalar one printed "old → new" in full; neither pointed at the change
// when a 400-character intro gained one word. The only difference left between
// them is that `lines` preserves its own newlines.
import { computed } from 'vue'
import TextDiff from './TextDiff.vue'
import type { EditFieldConfig } from './types'
import { diffItems, formatEditItem, formatEditValue } from './utils'

const props = defineProps<{
  label: string
  diffHint?: string
  from: unknown
  to: unknown
  config?: EditFieldConfig
}>()

// The scalar fallback diffs the DISPLAY form, not the raw value: an enum's
// label, a boolean's 是/否. Diffing `true` against `false` would tint the whole
// token anyway, and the reader never sees the raw value elsewhere.
const text = computed(() =>
  props.diffHint === 'lines'
    ? {
        from: typeof props.from === 'string' ? props.from : '',
        to: typeof props.to === 'string' ? props.to : '',
        preWrap: true
      }
    : {
        from: formatEditValue(props.from, props.config),
        to: formatEditValue(props.to, props.config),
        preWrap: false
      }
)

const items = computed(() =>
  props.diffHint === 'items' || props.diffHint === 'image'
    ? diffItems(props.from, props.to)
    : { added: [], removed: [], kept: [] }
)

const resolveImage = (value: unknown): string =>
  props.config?.resolveImage ? props.config.resolveImage(value) : ''

// Image hint covers both a scalar imagehash field and image lists.
const isImage = computed(() => props.diffHint === 'image')
const imagePairs = computed(() => {
  if (!isImage.value) {
    return { removed: [] as string[], added: [] as string[] }
  }
  if (Array.isArray(props.from) || Array.isArray(props.to)) {
    return {
      removed: items.value.removed.map(resolveImage).filter((u) => u !== ''),
      added: items.value.added.map(resolveImage).filter((u) => u !== '')
    }
  }
  return {
    removed: [resolveImage(props.from)].filter((u) => u !== ''),
    added: [resolveImage(props.to)].filter((u) => u !== '')
  }
})
</script>

<template>
  <div class="space-y-1">
    <p class="text-default-700 text-sm font-semibold">{{ label }}</p>

    <!-- image: side-by-side previews (falls back to text when unresolvable) -->
    <div v-if="isImage" class="grid grid-cols-2 gap-2">
      <div class="bg-danger-50 space-y-1 rounded p-2">
        <p class="text-danger-600 text-xs">修改前</p>
        <div v-if="imagePairs.removed.length" class="flex flex-wrap gap-1">
          <img
            v-for="(url, i) in imagePairs.removed"
            :key="i"
            :src="url"
            loading="lazy"
            class="max-h-24 rounded object-cover"
          />
        </div>
        <p v-else class="text-default-500 text-xs break-all">
          {{ formatEditValue(from, config) }}
        </p>
      </div>
      <div class="bg-success-50 space-y-1 rounded p-2">
        <p class="text-success-600 text-xs">修改后</p>
        <div v-if="imagePairs.added.length" class="flex flex-wrap gap-1">
          <img
            v-for="(url, i) in imagePairs.added"
            :key="i"
            :src="url"
            loading="lazy"
            class="max-h-24 rounded object-cover"
          />
        </div>
        <p v-else class="text-default-500 text-xs break-all">
          {{ formatEditValue(to, config) }}
        </p>
      </div>
    </div>

    <!-- items: added / removed chips -->
    <div v-else-if="diffHint === 'items'" class="space-y-1">
      <div v-if="items.removed.length" class="flex flex-wrap gap-1">
        <KunChip
          v-for="(item, i) in items.removed"
          :key="`del-${i}`"
          size="sm"
          variant="flat"
          color="danger"
        >
          − {{ formatEditItem(item, config) }}
        </KunChip>
      </div>
      <div v-if="items.added.length" class="flex flex-wrap gap-1">
        <KunChip
          v-for="(item, i) in items.added"
          :key="`add-${i}`"
          size="sm"
          variant="flat"
          color="success"
        >
          + {{ formatEditItem(item, config) }}
        </KunChip>
      </div>
      <p
        v-if="!items.added.length && !items.removed.length"
        class="text-default-400 text-xs"
      >
        仅顺序调整
      </p>
    </div>

    <!-- text: unified word-level diff (multi-line intros keep their newlines) -->
    <TextDiff v-else :from="text.from" :to="text.to" :pre-wrap="text.preWrap" />
  </div>
</template>
