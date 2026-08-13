<script setup lang="ts">
const props = defineProps<{
  galgame: GalgameDetail
}>()

const emits = defineEmits<{
  openRating: []
}>()

interface ExternalSourceMeta {
  label: string
  hint: string
}

const EXTERNAL_SOURCES: Record<string, ExternalSourceMeta> = {
  vndb: { label: 'VNDB', hint: 'VNDB 用户平均分, 满分 10' },
  bangumi: { label: 'Bangumi', hint: 'Bangumi 用户平均分, 满分 10' },
  erogamescape: {
    label: 'ErogameScape',
    hint: 'ErogameScape 中位数, 满分 100'
  },
  dlsite: { label: 'DLsite', hint: 'DLsite 星级, 满分 5' }
}

const SOURCE_ORDER = ['vndb', 'bangumi', 'erogamescape', 'dlsite']

const formatScore = (score: number) => String(Number(score.toFixed(2)))

const formatVotes = (count: number) => `${count.toLocaleString('en-US')} 人`

const externalTiles = computed(() => {
  const rows = props.galgame.external_ratings ?? []
  return SOURCE_ORDER.flatMap((source) => {
    const row = rows.find((r) => r.source === source)
    const meta = EXTERNAL_SOURCES[source]
    if (!row || !meta) return []
    return [{ ...row, ...meta }]
  })
})

const forumCount = computed(() => props.galgame.rating_count ?? 0)
const forumScore = computed(() => (props.galgame.rating ?? 0).toFixed(1))

const tileClass =
  'bg-default-100 flex min-w-24 shrink-0 flex-col gap-0.5 rounded-lg px-3 py-2'
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
      :class="cn(tileClass, 'hover:bg-default-200 text-left transition-colors')"
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

    <div v-else :class="tileClass">
      <span class="flex items-baseline gap-1">
        <KunIcon name="lucide:star" class="text-warning" />
        <span class="text-2xl leading-none font-semibold">
          {{ forumScore }}
        </span>
      </span>
      <KunTooltip text="本站用户评分, 满分 10 (贝叶斯平均)">
        <span class="text-default-500 text-xs">
          本站 · {{ formatVotes(forumCount) }}
        </span>
      </KunTooltip>
    </div>

    <div v-for="tile in externalTiles" :key="tile.source" :class="tileClass">
      <span class="flex items-baseline gap-1">
        <span class="text-2xl leading-none font-semibold">
          {{ formatScore(tile.score) }}
        </span>
        <span v-if="tile.rank" class="text-default-500 text-xs">
          #{{ tile.rank }}
        </span>
      </span>
      <KunTooltip :text="tile.hint">
        <span class="text-default-500 text-xs whitespace-nowrap">
          {{ tile.label }} · {{ formatVotes(tile.vote_count) }}
        </span>
      </KunTooltip>
    </div>
  </KunScrollShadow>
</template>
