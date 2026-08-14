<script setup lang="ts">
import {
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  type KunGalgameExternalRatingSource
} from '~/constants/galgame-rating'
import { externalRatingHistogram } from './_stats'

const props = defineProps<{
  galgame: GalgameDetail
  source: KunGalgameExternalRatingSource
}>()

const STAT_LABELS = [
  ['average', '平均分'],
  ['stdev', '标准差'],
  ['min', '最低分'],
  ['max', '最高分']
] as const

const meta = computed(() => KUN_GALGAME_EXTERNAL_RATING_MAP[props.source])

const row = computed(() =>
  props.galgame.external_ratings?.find((r) => r.source === props.source)
)

// Every source publishes something different — bangumi/dlsite a histogram,
// erogamescape spread stats, vndb neither. Read the capability off the payload
// instead of hardcoding it per source, so a source that gains data later just
// starts rendering.
const buckets = computed(() =>
  row.value?.distribution?.length
    ? externalRatingHistogram(row.value.distribution, meta.value.max)
    : null
)

const statRows = computed(() => {
  const stats = row.value?.stats
  if (!stats) {
    return []
  }
  return STAT_LABELS.flatMap(([key, label]) => {
    const value = stats[key]
    return value == null ? [] : [{ key, label, value }]
  })
})

// dlsite intentionally has no link template: the purchase URL is assembled
// server-side so it carries the affiliate id. No URL means no link, not a
// hand-built one that drops the affiliate.
const link = computed(() => {
  if (props.source === 'dlsite') {
    return props.galgame.dlsite_purchase_url ?? ''
  }
  const externalId = props.galgame.refs?.[props.source]
  return externalId ? (meta.value.link?.(externalId) ?? '') : ''
})
</script>

<template>
  <div v-if="row" class="space-y-5">
    <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
      <div>
        <div class="flex items-baseline gap-1">
          <span class="text-3xl leading-none font-semibold">
            {{ Number(row.score.toFixed(2)) }}
          </span>
          <span class="text-default-400 text-sm">/ {{ meta.max }}</span>
        </div>
        <span class="text-default-500 text-xs">{{ meta.hint }}</span>
      </div>

      <div class="text-default-500 flex gap-6 text-sm">
        <span>
          评分人数
          <span class="text-foreground font-medium tabular-nums">
            {{ row.vote_count.toLocaleString('en-US') }}
          </span>
        </span>
        <span v-if="row.rank">
          站内排名
          <span class="text-foreground font-medium tabular-nums">
            #{{ row.rank }}
          </span>
        </span>
      </div>
    </div>

    <div v-if="buckets" class="space-y-1">
      <h4 class="font-medium">评分分布</h4>
      <GalgameHeaderRatingDistributionChart
        :galgame-id="galgame.id"
        :source="source"
        :buckets="buckets"
        :mine="null"
        :unit="meta.unit"
      />
    </div>

    <div v-if="statRows.length" class="space-y-1.5">
      <h4 class="font-medium">分数概况</h4>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div
          v-for="stat in statRows"
          :key="stat.key"
          class="bg-default-100 rounded-lg px-3 py-2"
        >
          <div class="text-default-500 text-xs">{{ stat.label }}</div>
          <div class="font-medium tabular-nums">
            {{ Number(stat.value.toFixed(2)) }}
          </div>
        </div>
      </div>
      <p class="text-default-400 text-xs">
        这些数字和上面的分数出自同一批投票, 但上面那个是{{ meta.aggregate }},
        和这里的平均分不是同一个数。
      </p>
    </div>

    <GalgameHeaderRatingSourceCompare :galgame="galgame" :highlight="source" />

    <KunInfo
      v-if="!buckets && !statRows.length"
      color="default"
      title="外部来源只有汇总分"
      :description="`本站从百科同步到的只有 ${meta.label} 的平均分、评分人数和排名, 拿不到每一条评分具体是谁打的, 所以这里没有分布图和评分人列表。`"
    />
    <KunInfo
      v-else
      color="default"
      title="外部来源没有逐人评分"
      :description="`本站从百科同步到的是 ${meta.label} 的汇总统计, 拿不到每一条评分具体是谁打的, 所以这里没有评分人列表。`"
    />

    <KunButton
      v-if="link"
      :href="link"
      target="_blank"
      rel="noopener noreferrer"
      variant="flat"
      color="primary"
      size="sm"
    >
      <span class="flex items-center gap-1">
        <KunIcon name="lucide:external-link" />
        在 {{ meta.label }} 查看
      </span>
    </KunButton>
  </div>
</template>
