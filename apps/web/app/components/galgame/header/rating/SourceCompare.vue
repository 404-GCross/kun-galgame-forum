<script setup lang="ts">
import {
  KUN_GALGAME_EXTERNAL_RATING_CONST,
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  KUN_GALGAME_LOCAL_RATING_META,
  KUN_GALGAME_LOCAL_RATING_SOURCE
} from '~/constants/galgame-rating'

const props = defineProps<{
  galgame: GalgameDetail
  highlight: string
}>()

const rows = computed(() => {
  const local = props.galgame.rating_count
    ? [
        {
          key: KUN_GALGAME_LOCAL_RATING_SOURCE,
          label: KUN_GALGAME_LOCAL_RATING_META.label,
          max: KUN_GALGAME_LOCAL_RATING_META.max,
          score: props.galgame.rating ?? 0,
          count: props.galgame.rating_count
        }
      ]
    : []

  const external = KUN_GALGAME_EXTERNAL_RATING_CONST.flatMap((source) => {
    const row = props.galgame.external_ratings?.find((r) => r.source === source)
    if (!row) return []
    const meta = KUN_GALGAME_EXTERNAL_RATING_MAP[source]
    return [
      {
        key: source as string,
        label: meta.label,
        max: meta.max,
        score: row.score,
        count: row.vote_count
      }
    ]
  })

  return [...local, ...external]
})

const normalized = (score: number, max: number) => (score / max) * 10
</script>

<template>
  <div v-if="rows.length > 1" class="space-y-2">
    <div class="flex items-baseline justify-between">
      <h4 class="font-medium">各来源对比</h4>
      <span class="text-default-500 text-xs">统一折算为满分 10</span>
    </div>

    <div
      v-for="row in rows"
      :key="row.key"
      :class="
        cn(
          'grid grid-cols-[7rem_1fr_auto] items-center gap-3 rounded-lg px-2 py-1.5',
          row.key === highlight && 'bg-default-100'
        )
      "
    >
      <span
        :class="
          cn(
            'truncate text-xs',
            row.key === highlight ? 'font-medium' : 'text-default-500'
          )
        "
      >
        {{ row.label }}
      </span>
      <KunProgress
        :value="normalized(row.score, row.max)"
        :max="10"
        size="sm"
        :color="row.key === highlight ? 'warning' : 'primary'"
      />
      <span class="text-default-500 shrink-0 text-xs tabular-nums">
        {{ normalized(row.score, row.max).toFixed(1) }}
        <span class="text-default-400">
          · {{ row.count.toLocaleString('en-US') }} 人
        </span>
      </span>
    </div>
  </div>
</template>
