<script setup lang="ts">
import {
  KUN_GALGAME_EXTERNAL_RATING_CONST,
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  KUN_GALGAME_LOCAL_RATING_META,
  KUN_GALGAME_LOCAL_RATING_SOURCE
} from '~/constants/galgame-rating'

const props = defineProps<{
  galgame: GalgameDetail
}>()

const emits = defineEmits<{
  openRating: []
  openDetail: [string]
}>()

const formatVotes = (count: number) => `${count.toLocaleString('en-US')} 人`

const forumCount = computed(() => props.galgame.rating_count ?? 0)

const tiles = computed(() => {
  const local = forumCount.value
    ? [
        {
          key: KUN_GALGAME_LOCAL_RATING_SOURCE,
          icon: 'lucide:star',
          score: KUN_GALGAME_LOCAL_RATING_META.formatScore(
            props.galgame.rating ?? 0
          ),
          suffix: KUN_GALGAME_LOCAL_RATING_META.scoreSuffix,
          caption: `${KUN_GALGAME_LOCAL_RATING_META.label} · ${formatVotes(forumCount.value)}`,
          tooltip: `${KUN_GALGAME_LOCAL_RATING_META.hint}, ${KUN_GALGAME_LOCAL_RATING_META.scale}`
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
        icon: '',
        score: meta.formatScore(row.score),
        suffix: meta.scoreSuffix,
        caption: `${meta.label} · ${formatVotes(row.vote_count)}`,
        tooltip: `${meta.hint}, ${meta.scale}`
      }
    ]
  })

  return [...local, ...external]
})

const tileClass =
  'bg-default-100 hover:bg-default-200 flex h-full min-w-24 shrink-0 flex-col gap-0.5 rounded-lg px-3 py-2 text-left transition-colors'
</script>

<template>
  <KunScrollShadow
    axis="horizontal"
    shadow-size="1.5rem"
    :wheel="true"
    content-class="flex items-stretch gap-2"
  >
    <button
      v-if="!forumCount"
      type="button"
      :class="tileClass"
      @click="emits('openRating')"
    >
      <span
        class="text-default-600 flex items-center gap-1 text-sm font-medium"
      >
        <KunIcon name="lucide:star" class="text-warning" />
        还没有评分
      </span>
      <span class="text-primary text-xs">来评第一个</span>
    </button>

    <KunTooltip
      v-for="tile in tiles"
      :key="tile.key"
      :text="tile.tooltip"
      class-name="shrink-0"
    >
      <button
        type="button"
        :class="tileClass"
        @click="emits('openDetail', tile.key)"
      >
        <span class="flex items-baseline gap-1">
          <KunIcon v-if="tile.icon" :name="tile.icon" class="text-warning" />
          <span class="flex items-baseline gap-0.5">
            <span class="text-2xl leading-none font-semibold tabular-nums">
              {{ tile.score }}
            </span>
            <span class="text-default-500 text-sm leading-none font-medium">
              {{ tile.suffix }}
            </span>
          </span>
        </span>
        <span class="text-default-500 text-xs whitespace-nowrap">
          {{ tile.caption }}
        </span>
      </button>
    </KunTooltip>
  </KunScrollShadow>
</template>
