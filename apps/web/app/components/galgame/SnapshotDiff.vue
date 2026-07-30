<script setup lang="ts">
// Field diff renderer for wiki galgame snapshots.
//
// Three regimes, picked by value shape:
//
//   1. SCALAR             — string / number / boolean.
//      One UNIFIED block: the text once, with only the changed runs
//      tinted (EditkitTextDiff, word-level via jsdiff). This replaced a
//      2-column 原版/新版 layout that printed the whole field twice and
//      tinted one side red, the other green — which cannot answer "what
//      changed": a one-word edit inside a 400-character intro was
//      invisible between two walls of coloured text. LARGE_TEXT_FIELDS
//      render pre-wrap so multi-line intros keep their line structure.
//
//   2. ARRAY OF OBJECTS   — covers / screenshots / links.
//      Still 2-column, and rightly so: these are images and JSON rows,
//      not prose, and comparing a changed cover means seeing both.
//      LEFT  = removed items + OLD version of changed items.
//      RIGHT = added items + NEW version of changed items.
//      Changed items appear in the same logical row on both sides so
//      the user can compare attributes line-by-line.
//
//   3. ARRAY OF SCALARS   — tag_ids / official_ids / engine_ids / aliases.
//      LEFT  = removed values as "-" chips.
//      RIGHT = added values as "+" chips.
//
// On mobile (<md) the two columns stack vertically (old on top, new
// below) since side-by-side at narrow viewports is unreadable.
import { KUN_GALGAME_RESOURCE_PULL_REQUEST_I18N_FIELD_MAP } from '~/constants/galgame'

// Wiki ships a `names` map alongside the diff (K-PR, 2026-Q2):
//   { tags, officials, engines, series } each a {id → displayName}
// dict scoped to the IDs referenced by THIS particular diff. We use it
// to swap raw numeric ids for human-readable names inline — no N+1
// follow-up requests needed.
//
// If the dict is undefined (older wiki version not yet shipping names)
// or the id is missing (entity was deleted between revisions), we fall
// back to "已删除 #<id>" per the wiki integration callout.
type NamesDict = {
  tags?: Record<string, string>
  officials?: Record<string, string>
  engines?: Record<string, string>
  series?: Record<string, string>
}

const props = defineProps<{
  changedKeys: Record<string, boolean>
  oldSnap: Record<string, unknown>
  newSnap: Record<string, unknown>
  // Optional: when absent, raw ids render through unchanged. Keeps the
  // component backward-compatible with older wiki responses.
  names?: NamesDict
}>()

const label = (key: string): string => {
  const m = KUN_GALGAME_RESOURCE_PULL_REQUEST_I18N_FIELD_MAP[key]
  return typeof m === 'string' ? m : key
}

// Field key → which dict under `names` we read. Anything not in this
// map renders the raw value (e.g. aliases are scalar strings, not ids).
const FIELD_NAMES_KEY: Record<string, keyof NamesDict> = {
  tag_ids: 'tags',
  official_ids: 'officials',
  engine_ids: 'engines',
  series_id: 'series'
}

const lookupName = (fieldKey: string, value: unknown): string => {
  const dictKey = FIELD_NAMES_KEY[fieldKey]
  if (!dictKey) return String(value ?? '')
  if (value === undefined || value === null || value === '') return ''
  // Two distinct "no name" cases that must NOT collapse:
  //   1. `names` prop absent entirely (older wiki, or no resolver
  //      shipped with the diff). Backward-compat: render the raw id.
  //   2. `names` IS provided but the id is missing from it ⇒ entity
  //      deleted between revisions. Render "已删除 #<id>".
  if (!props.names) return String(value)
  const dict = props.names[dictKey]
  if (!dict) return String(value)
  const name = dict[String(value)]
  return name ?? `已删除 #${value}`
}

const arrayObjectConfig: Record<
  string,
  { keyFn: (item: Record<string, unknown>) => string; compareFields?: string[] }
> = {
  covers: {
    keyFn: (c) => String(c['image_hash'] ?? ''),
    compareFields: ['sort_order', 'sexual', 'violence']
  },
  screenshots: {
    keyFn: (s) => String(s['image_hash'] ?? ''),
    compareFields: ['sort_order', 'caption', 'sexual', 'violence']
  },
  links: {
    keyFn: (l) => `${l['name']}|||${l['link']}`,
    compareFields: []
  }
}

const SCALAR_ARRAY_FIELDS = new Set([
  'tag_ids',
  'official_ids',
  'engine_ids',
  'aliases'
])

const LARGE_TEXT_FIELDS = new Set([
  'intro_en_us',
  'intro_ja_jp',
  'intro_zh_cn',
  'intro_zh_tw',
  'description'
])

type RowKind = 'scalar' | 'array_obj' | 'array_scalar'

interface Row {
  key: string
  label: string
  kind: RowKind
  // SCALAR: the raw display text of each side; TextDiff does the rest.
  scalar?: { old: string; new: string; preWrap: boolean }
  // ARRAY OF OBJECTS — pre-built side breakdown.
  arrayObj?: {
    added: Record<string, unknown>[]
    removed: Record<string, unknown>[]
    changed: {
      key: string
      old: Record<string, unknown>
      new: Record<string, unknown>
      fields: string[]
    }[]
  }
  arrayScalar?: { added: (string | number)[]; removed: (string | number)[] }
}

const renderScalar = (
  key: string,
  oldVal: unknown,
  newVal: unknown
): Row['scalar'] => {
  // For id-bearing scalar fields (currently just series_id), display
  // the resolved entity NAME on each side rather than the raw id —
  // matches what we do for array-of-ids fields below.
  const o = FIELD_NAMES_KEY[key]
    ? lookupName(key, oldVal)
    : oldVal === undefined || oldVal === null
      ? ''
      : String(oldVal)
  const n = FIELD_NAMES_KEY[key]
    ? lookupName(key, newVal)
    : newVal === undefined || newVal === null
      ? ''
      : String(newVal)
  // Plain strings out — TextDiff renders text nodes, so nothing here has to be
  // HTML-escaped the way the old string-building renderer did.
  return { old: o, new: n, preWrap: LARGE_TEXT_FIELDS.has(key) }
}

const rows = computed<Row[]>(() => {
  const out: Row[] = []
  for (const key of Object.keys(props.changedKeys)) {
    if (!props.changedKeys[key]) continue
    const o = props.oldSnap?.[key]
    const n = props.newSnap?.[key]

    // ARRAY OF OBJECTS
    if (arrayObjectConfig[key]) {
      const cfg = arrayObjectConfig[key]!
      const oArr = Array.isArray(o) ? (o as Record<string, unknown>[]) : []
      const nArr = Array.isArray(n) ? (n as Record<string, unknown>[]) : []
      const d = arrayDiffByKey(
        oArr,
        nArr,
        cfg.keyFn,
        cfg.compareFields as never
      )
      if (d.added.length || d.removed.length || d.changed.length) {
        out.push({
          key,
          label: label(key),
          kind: 'array_obj',
          arrayObj: d
        })
      }
      continue
    }

    // ARRAY OF SCALARS
    if (SCALAR_ARRAY_FIELDS.has(key)) {
      const oArr = Array.isArray(o) ? (o as (string | number)[]) : []
      const nArr = Array.isArray(n) ? (n as (string | number)[]) : []
      const d = arrayDiffScalar(oArr, nArr)
      if (d.added.length || d.removed.length) {
        out.push({
          key,
          label: label(key),
          kind: 'array_scalar',
          arrayScalar: d
        })
      }
      continue
    }

    // RUNTIME FALLBACK
    if (Array.isArray(o) || Array.isArray(n)) {
      const oArr = Array.isArray(o) ? o : []
      const nArr = Array.isArray(n) ? n : []
      const allScalars = [...oArr, ...nArr].every(
        (v) => typeof v !== 'object' || v === null
      )
      if (allScalars) {
        const d = arrayDiffScalar(
          oArr as (string | number)[],
          nArr as (string | number)[]
        )
        if (d.added.length || d.removed.length) {
          out.push({
            key,
            label: label(key),
            kind: 'array_scalar',
            arrayScalar: d
          })
        }
        continue
      }
    }

    const scalar = renderScalar(key, o, n)
    if (scalar && (scalar.old.length > 0 || scalar.new.length > 0)) {
      out.push({ key, label: label(key), kind: 'scalar', scalar })
    }
  }
  return out
})

const inlineRow = (item: Record<string, unknown>): string =>
  JSON.stringify(item)

// Image-valued array fields (covers / screenshots) render the picture itself —
// far more legible than a raw JSON row when a cover changes.
const IMAGE_FIELDS = new Set(['covers', 'screenshots'])
const isImageRow = (key: string): boolean => IMAGE_FIELDS.has(key)
const imageOf = (item: Record<string, unknown>): string => {
  const hash = item['image_hash']
  return hash ? imageHashUrl(imageCdnBase(), String(hash), 'mini') : ''
}
</script>

<template>
  <div class="space-y-5">
    <KunNull v-if="!rows.length" description="无字段变化" />

    <p v-else class="text-default-500 text-xs">
      只标出改动的部分：
      <del
        class="bg-danger/15 text-danger-600 decoration-danger-600/50 rounded px-1"
        >删除</del
      >
      <ins class="bg-success/15 text-success-600 ml-1 rounded px-1 no-underline"
        >新增</ins
      >
    </p>

    <div v-for="row in rows" :key="row.key" class="space-y-2">
      <!-- Field label -->
      <p class="text-default-700 text-sm font-semibold">{{ row.label }}</p>

      <!-- SCALAR — one unified block, only the changed runs tinted -->
      <EditkitTextDiff
        v-if="row.kind === 'scalar' && row.scalar"
        :from="row.scalar.old"
        :to="row.scalar.new"
        :pre-wrap="row.scalar.preWrap"
      />

      <!-- ARRAYS — 2-col side-by-side (images / JSON rows, not prose) -->
      <div v-else class="grid grid-cols-1 gap-2 md:grid-cols-2">
        <!-- ─────── LEFT: 原版 ─────── -->
        <div
          class="bg-danger-50/40 border-danger-200/60 dark:bg-danger-500/5 dark:border-danger-500/20 rounded-lg border"
        >
          <div
            class="text-default-600 border-danger-200/60 dark:border-danger-500/20 flex items-center gap-1.5 border-b px-3 py-1.5 text-xs font-medium"
          >
            <KunIcon
              name="lucide:minus-circle"
              class="text-danger-500 h-3.5 w-3.5"
            />
            原版
          </div>
          <div class="px-3 py-2 text-sm">
            <!-- ARRAY OF SCALARS — removed items.
                 lookupName() swaps raw ids for display names when
                 row.key is an id-bearing field (tag_ids etc.);
                 returns raw value otherwise (aliases). -->
            <div
              v-if="row.kind === 'array_scalar' && row.arrayScalar"
              class="flex flex-wrap gap-1.5"
            >
              <KunChip
                v-for="v in row.arrayScalar.removed"
                :key="`-${v}`"
                color="danger"
                size="sm"
                variant="flat"
              >
                - {{ lookupName(row.key, v) }}
              </KunChip>
              <span
                v-if="!row.arrayScalar.removed.length"
                class="text-default-400 text-xs italic"
              >
                无项移除
              </span>
            </div>

            <!-- ARRAY OF OBJECTS — removed + old of changed -->
            <div
              v-else-if="row.kind === 'array_obj' && row.arrayObj"
              class="space-y-2"
            >
              <div
                v-for="(ch, i) in row.arrayObj.changed"
                :key="`ch-old-${i}`"
                class="space-y-1"
              >
                <div class="flex items-center gap-1.5">
                  <KunChip color="warning" size="sm" variant="flat"
                    >修改</KunChip
                  >
                  <span class="text-default-500 text-xs">
                    {{ ch.fields.join(' / ') }}
                  </span>
                </div>
                <img
                  v-if="isImageRow(row.key)"
                  :src="imageOf(ch.old)"
                  loading="lazy"
                  class="border-default-200 max-h-32 max-w-full rounded-md border object-contain"
                />
                <code v-else class="text-default-600 block text-xs break-all">
                  {{ inlineRow(ch.old) }}
                </code>
              </div>
              <div
                v-for="(item, i) in row.arrayObj.removed"
                :key="`rm-${i}`"
                class="space-y-1"
              >
                <KunChip color="danger" size="sm" variant="flat"
                  >- 删除</KunChip
                >
                <img
                  v-if="isImageRow(row.key)"
                  :src="imageOf(item)"
                  loading="lazy"
                  class="border-default-200 max-h-32 max-w-full rounded-md border object-contain"
                />
                <code v-else class="text-default-600 block text-xs break-all">
                  {{ inlineRow(item) }}
                </code>
              </div>
              <span
                v-if="
                  !row.arrayObj.removed.length && !row.arrayObj.changed.length
                "
                class="text-default-400 text-xs italic"
              >
                无项移除
              </span>
            </div>
          </div>
        </div>

        <!-- ─────── RIGHT: 新版 ─────── -->
        <div
          class="bg-success-50/40 border-success-200/60 dark:bg-success-500/5 dark:border-success-500/20 rounded-lg border"
        >
          <div
            class="text-default-600 border-success-200/60 dark:border-success-500/20 flex items-center gap-1.5 border-b px-3 py-1.5 text-xs font-medium"
          >
            <KunIcon
              name="lucide:plus-circle"
              class="text-success-500 h-3.5 w-3.5"
            />
            新版
          </div>
          <div class="px-3 py-2 text-sm">
            <!-- ARRAY OF SCALARS — added items.
                 Same lookupName treatment as the LEFT column. -->
            <div
              v-if="row.kind === 'array_scalar' && row.arrayScalar"
              class="flex flex-wrap gap-1.5"
            >
              <KunChip
                v-for="v in row.arrayScalar.added"
                :key="`+${v}`"
                color="success"
                size="sm"
                variant="flat"
              >
                + {{ lookupName(row.key, v) }}
              </KunChip>
              <span
                v-if="!row.arrayScalar.added.length"
                class="text-default-400 text-xs italic"
              >
                无项新增
              </span>
            </div>

            <!-- ARRAY OF OBJECTS — added + new of changed -->
            <div
              v-else-if="row.kind === 'array_obj' && row.arrayObj"
              class="space-y-2"
            >
              <div
                v-for="(ch, i) in row.arrayObj.changed"
                :key="`ch-new-${i}`"
                class="space-y-1"
              >
                <div class="flex items-center gap-1.5">
                  <KunChip color="warning" size="sm" variant="flat"
                    >修改</KunChip
                  >
                  <span class="text-default-500 text-xs">
                    {{ ch.fields.join(' / ') }}
                  </span>
                </div>
                <img
                  v-if="isImageRow(row.key)"
                  :src="imageOf(ch.new)"
                  loading="lazy"
                  class="border-default-200 max-h-32 max-w-full rounded-md border object-contain"
                />
                <code v-else class="text-default-700 block text-xs break-all">
                  {{ inlineRow(ch.new) }}
                </code>
              </div>
              <div
                v-for="(item, i) in row.arrayObj.added"
                :key="`add-${i}`"
                class="space-y-1"
              >
                <KunChip color="success" size="sm" variant="flat"
                  >+ 新增</KunChip
                >
                <img
                  v-if="isImageRow(row.key)"
                  :src="imageOf(item)"
                  loading="lazy"
                  class="border-default-200 max-h-32 max-w-full rounded-md border object-contain"
                />
                <code v-else class="text-default-700 block text-xs break-all">
                  {{ inlineRow(item) }}
                </code>
              </div>
              <span
                v-if="
                  !row.arrayObj.added.length && !row.arrayObj.changed.length
                "
                class="text-default-400 text-xs italic"
              >
                无项新增
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
