<script setup lang="ts">
const props = defineProps<{
  data: GalgameCalendarMonth
}>()

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

type DayStatus = 'past' | 'today' | 'future'

interface DayGroup {
  key: string
  day: number
  weekday: string
  status: DayStatus
  isMonthBucket: boolean
  games: GalgameCard[]
}

// Items arrive pre-sorted by the wiki (date asc; within a day, exact-day
// entries before the "日未定" month-precision tail). Split them into per-day
// groups plus one trailing "日期待定" bucket for month-precision entries; a Map
// keeps insertion order so the day groups stay date-ascending. status (vs the
// JST `today`) drives the date-tile colour: released days muted, today filled,
// upcoming outlined.
const groups = computed<DayGroup[]>(() => {
  const today = props.data.today
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
    const [y, mo, d] = key.split('-').map(Number)
    const status: DayStatus =
      key === today ? 'today' : key < today ? 'past' : 'future'
    out.push({
      key,
      day: d ?? 0,
      // Fixed calendar components → the weekday is timezone-invariant, so SSR
      // and client agree (no hydration mismatch).
      weekday: WEEKDAYS[new Date(y ?? 0, (mo ?? 1) - 1, d ?? 1).getDay()] ?? '',
      status,
      isMonthBucket: false,
      games
    })
  }
  if (monthBucket.length) {
    out.push({
      key: '__month-bucket__',
      day: 0,
      weekday: '',
      status: 'future',
      isMonthBucket: true,
      games: monthBucket
    })
  }
  return out
})

const tileClass = (grp: DayGroup) => {
  if (!grp.isMonthBucket && grp.status === 'today') {
    return 'bg-primary text-white'
  }
  if (!grp.isMonthBucket && grp.status === 'future') {
    return 'border-primary text-primary border-2'
  }
  return 'bg-default-100 text-default-500'
}
</script>

<template>
  <div class="flex flex-col">
    <div
      v-for="(grp, i) in groups"
      :key="grp.key"
      class="flex flex-col gap-3 py-5 sm:flex-row sm:gap-5"
      :class="{ 'border-default-200 border-t': i > 0 }"
    >
      <!-- Date tile (calendar-style): big day number + weekday. Today filled,
           upcoming outlined, released muted. On mobile it's an inline header
           row above the cards; on desktop a fixed left column. -->
      <div
        class="flex shrink-0 items-center gap-3 sm:w-16 sm:flex-col sm:gap-1.5"
      >
        <div
          class="flex size-14 flex-col items-center justify-center rounded-xl"
          :class="tileClass(grp)"
        >
          <KunIcon
            v-if="grp.isMonthBucket"
            name="lucide:calendar-clock"
            class="size-6"
          />
          <span v-else class="text-3xl font-bold leading-none">
            {{ grp.day }}
          </span>
        </div>

        <div class="flex items-center gap-2 sm:flex-col sm:gap-1">
          <span class="text-default-500 text-sm">
            {{ grp.isMonthBucket ? '日期待定' : grp.weekday }}
          </span>
          <KunChip
            v-if="!grp.isMonthBucket && grp.status === 'today'"
            variant="solid"
            color="primary"
            size="sm"
          >
            今日
          </KunChip>
        </div>
      </div>

      <div class="flex-1">
        <GalgameCard :galgames="grp.games" />
      </div>
    </div>
  </div>
</template>
