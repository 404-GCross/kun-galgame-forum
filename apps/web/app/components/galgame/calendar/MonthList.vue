<script setup lang="ts">
// The whole month as date-tile day-rows (the earlier UI-pass style): a left
// date tile (big day number + weekday; today filled, upcoming outlined,
// released muted) beside that day's cards. Today's row is pinned to the top,
// then the remaining days run chronologically, then the 日期待定 bucket. Each
// row carries an id so the calendar grid can scroll-to it on click.
const props = defineProps<{
  data: GalgameCalendarMonth
}>()

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

type DayStatus = 'past' | 'today' | 'future'

interface DayRow {
  key: string
  day: number
  weekday: string
  status: DayStatus
  isBucket: boolean
  games: GalgameCard[]
}

const rows = computed<DayRow[]>(() => {
  const today = props.data.today
  const monthCmp = props.data.month.localeCompare(today.slice(0, 7))
  const todayDay = monthCmp === 0 ? Number(today.slice(8, 10)) : -1

  const dayMap = new Map<number, GalgameCard[]>()
  const bucket: GalgameCard[] = []
  for (const game of props.data.items) {
    if (game.releasePrecision === 'month' || !game.releaseDate) {
      bucket.push(game)
      continue
    }
    const d = Number(game.releaseDate.slice(8, 10))
    if (!dayMap.has(d)) {
      dayMap.set(d, [])
    }
    dayMap.get(d)!.push(game)
  }

  const [y, mo] = props.data.month.split('-').map(Number)
  const statusOf = (d: number): DayStatus =>
    monthCmp < 0
      ? 'past'
      : monthCmp > 0
        ? 'future'
        : d === todayDay
          ? 'today'
          : d < todayDay
            ? 'past'
            : 'future'
  const mkRow = (d: number): DayRow => ({
    key: String(d),
    day: d,
    // Fixed calendar components → timezone-invariant weekday (SSR-safe).
    weekday: WEEKDAYS[new Date(y ?? 0, (mo ?? 1) - 1, d).getDay()] ?? '',
    status: statusOf(d),
    isBucket: false,
    games: dayMap.get(d)!
  })

  const days = [...dayMap.keys()].sort((a, b) => a - b)
  const out: DayRow[] = []
  // Today pinned at the top, then the rest of the month chronologically.
  if (todayDay > 0 && dayMap.has(todayDay)) {
    out.push(mkRow(todayDay))
  }
  for (const d of days) {
    if (d !== todayDay) {
      out.push(mkRow(d))
    }
  }
  if (bucket.length) {
    out.push({
      key: 'bucket',
      day: 0,
      weekday: '',
      status: 'future',
      isBucket: true,
      games: bucket
    })
  }
  return out
})

const tileClass = (r: DayRow) => {
  if (!r.isBucket && r.status === 'today') {
    return 'bg-primary text-white'
  }
  if (!r.isBucket && r.status === 'future') {
    return 'border-primary text-primary border-2'
  }
  return 'bg-default-100 text-default-500'
}
</script>

<template>
  <div class="flex flex-col">
    <div
      v-for="(r, i) in rows"
      :id="`cal-day-${r.key}`"
      :key="r.key"
      class="flex scroll-mt-20 flex-col gap-3 py-5 sm:flex-row sm:gap-5"
      :class="{ 'border-default-200 border-t': i > 0 }"
    >
      <!-- Date tile: today filled, upcoming outlined, released muted. -->
      <div
        class="flex shrink-0 items-center gap-3 sm:w-16 sm:flex-col sm:gap-1.5"
      >
        <div
          class="flex size-14 flex-col items-center justify-center rounded-xl"
          :class="tileClass(r)"
        >
          <KunIcon
            v-if="r.isBucket"
            name="lucide:calendar-clock"
            class="size-6"
          />
          <span v-else class="text-3xl font-bold leading-none">{{ r.day }}</span>
        </div>

        <div class="flex items-center gap-2 sm:flex-col sm:gap-1">
          <span class="text-default-500 text-sm">
            {{ r.isBucket ? '日期待定' : r.weekday }}
          </span>
          <KunChip
            v-if="!r.isBucket && r.status === 'today'"
            variant="solid"
            color="primary"
            size="sm"
          >
            今日
          </KunChip>
        </div>
      </div>

      <div class="min-w-0 flex-1">
        <GalgameCard :galgames="r.games" />
      </div>
    </div>
  </div>
</template>
