<script setup lang="ts">
import {
  KUN_GALGAME_PLAYTIME_SOURCE_CONST,
  KUN_GALGAME_PLAYTIME_SOURCE_MAP
} from '~/constants/galgame-playtime'

const props = defineProps<{
  playtimes?: GalgamePlaytime[]
}>()

const chips = computed(() =>
  KUN_GALGAME_PLAYTIME_SOURCE_CONST.flatMap((source) => {
    const row = props.playtimes?.find((p) => p.source === source)
    if (!row) return []

    const meta = KUN_GALGAME_PLAYTIME_SOURCE_MAP[source]
    if (meta.hasVoteCount && row.vote_count < meta.minVotes) return []

    const duration = formatDurationMinutes(row.minutes)
    if (!duration) return []

    const votes = meta.hasVoteCount
      ? `${row.vote_count.toLocaleString('en-US')} 人`
      : ''
    return [
      {
        key: source,
        short: meta.short,
        duration,
        votes,
        tooltip: votes ? `${meta.hint}, 共 ${votes}` : meta.hint
      }
    ]
  })
)
</script>

<template>
  <div v-if="chips.length" class="flex flex-wrap items-center gap-2">
    <span class="text-default-600 flex shrink-0 items-center gap-1.5 text-sm">
      <KunIcon name="lucide:clock" class="text-default-400" />
      游玩时长
    </span>

    <KunTooltip
      v-for="chip in chips"
      :key="chip.key"
      :text="chip.tooltip"
      class-name="shrink-0"
    >
      <KunChip size="sm" variant="flat" color="default">
        <span class="text-default-500 font-medium">{{ chip.short }}</span>
        <span class="tabular-nums">{{ chip.duration }}</span>
        <span v-if="chip.votes" class="text-default-400 tabular-nums">
          · {{ chip.votes }}
        </span>
      </KunChip>
    </KunTooltip>
  </div>
</template>
