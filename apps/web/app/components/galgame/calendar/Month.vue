<script setup lang="ts">
const props = defineProps<{
  data: GalgameCalendarMonth
}>()

interface DayGroup {
  key: string
  label: string
  isToday: boolean
  isMonthBucket: boolean
  games: GalgameCard[]
}

// Items arrive pre-sorted by the wiki (date asc; within a day, exact-day
// entries before the "日未定" month-precision tail). We split them into
// per-day groups plus a single "本月内 · 日期待定" bucket for month-precision
// entries (whose day is unknown), rendered last. A Map preserves insertion
// order, so the day groups stay date-ascending without re-sorting.
const groups = computed<DayGroup[]>(() => {
  const dayMap = new Map<string, GalgameCard[]>()
  const monthBucket: GalgameCard[] = []

  for (const game of props.data.items) {
    if (game.releasePrecision === 'month' || !game.releaseDate) {
      monthBucket.push(game)
      continue
    }
    const key = game.releaseDate // YYYY-MM-DD
    if (!dayMap.has(key)) {
      dayMap.set(key, [])
    }
    dayMap.get(key)!.push(game)
  }

  const out: DayGroup[] = []
  for (const [key, games] of dayMap) {
    const [, mo, day] = key.split('-')
    out.push({
      key,
      label: `${Number(mo)} 月 ${Number(day)} 日`,
      isToday: key === props.data.today,
      isMonthBucket: false,
      games
    })
  }
  if (monthBucket.length) {
    out.push({
      key: '__month-bucket__',
      label: '本月内 · 日期待定',
      isToday: false,
      isMonthBucket: true,
      games: monthBucket
    })
  }
  return out
})
</script>

<template>
  <div class="flex flex-col gap-5">
    <section v-for="grp in groups" :key="grp.key" class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <KunIcon
          :name="
            grp.isMonthBucket ? 'lucide:calendar-clock' : 'lucide:calendar-days'
          "
          class="text-default-500"
        />
        <h3 class="font-medium">{{ grp.label }}</h3>
        <KunChip v-if="grp.isToday" variant="solid" color="primary" size="sm">
          今日
        </KunChip>
        <span class="text-default-400 text-sm">{{ grp.games.length }} 部</span>
      </div>

      <GalgameCard :galgames="grp.games" />
    </section>
  </div>
</template>
