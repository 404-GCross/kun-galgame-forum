<script setup lang="ts">
import {
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  type KunGalgameExternalRatingSource
} from '~/constants/galgame-rating'

const props = defineProps<{
  galgame: GalgameDetail
  source: KunGalgameExternalRatingSource
}>()

const meta = computed(() => KUN_GALGAME_EXTERNAL_RATING_MAP[props.source])

const row = computed(() =>
  props.galgame.external_ratings?.find((r) => r.source === props.source)
)

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

    <GalgameHeaderRatingSourceCompare :galgame="galgame" :highlight="source" />

    <KunInfo
      color="default"
      title="外部来源只有汇总分"
      :description="`本站从百科同步到的只有 ${meta.label} 的平均分、评分人数和排名, 拿不到每个用户各自打了几分, 所以这里没有分布图和评分人列表。`"
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
