<script setup lang="ts">
// The month view's right-hand content: a 3-month window (prev / focus / next)
// as date-tile day-rows, grouped into month sections. The parent centers the
// focus section in a scroll panel (the "wheel"), so prev peeks above + next
// below. Each tile shows its month above the day number (the section header
// scrolls out of view as you wheel); the focus section is marked for centering.
const props = defineProps<{
  months: GalgameCalendarUpcomingMonth[]
  today: string
  focusMonth: string
}>()

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

type DayStatus = 'past' | 'today' | 'future'

interface DayRow {
  id: string
  day: number
  monthShort: string
  weekday: string
  status: DayStatus
  isBucket: boolean
  games: GalgameCard[]
}

interface MonthSection {
  month: string
  label: string
  isFocus: boolean
  count: number
  rows: DayRow[]
}

const sections = computed<MonthSection[]>(() =>
  props.months.map((m) => {
    const [y, mo] = m.month.split('-').map(Number)
    const monthShort = `${mo} 月`
    const monthCmp = m.month.localeCompare(props.today.slice(0, 7))
    const todayDay = monthCmp === 0 ? Number(props.today.slice(8, 10)) : -1

    const dayMap = new Map<number, GalgameCard[]>()
    const bucket: GalgameCard[] = []
    for (const game of m.items) {
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

    const rows: DayRow[] = [...dayMap.keys()]
      .sort((a, b) => a - b)
      .map((d) => ({
        id: `${m.month}-${String(d).padStart(2, '0')}`,
        day: d,
        monthShort,
        // Fixed calendar components → timezone-invariant weekday (SSR-safe).
        weekday: WEEKDAYS[new Date(y ?? 0, (mo ?? 1) - 1, d).getDay()] ?? '',
        status: statusOf(d),
        isBucket: false,
        games: dayMap.get(d)!
      }))
    if (bucket.length) {
      rows.push({
        id: `${m.month}-bucket`,
        day: 0,
        monthShort,
        weekday: '',
        status: 'future',
        isBucket: true,
        games: bucket
      })
    }

    return {
      month: m.month,
      label: `${y} 年 ${mo} 月`,
      isFocus: m.month === props.focusMonth,
      count: m.items.length,
      rows
    }
  })
)

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
  <div class="flex flex-col gap-6">
    <section
      v-for="sec in sections"
      :key="sec.month"
      :data-focus-month="sec.isFocus ? '' : undefined"
      class="flex flex-col"
    >
      <div
        class="border-default-200 mb-1 flex items-center gap-2 border-b pb-1"
      >
        <h3
          :class="[
            'font-medium',
            sec.isFocus ? 'text-primary' : 'text-default-600'
          ]"
        >
          {{ sec.label }}
        </h3>
        <KunChip v-if="sec.isFocus" variant="flat" color="primary" size="sm">
          本月
        </KunChip>
        <span class="text-default-400 text-sm">{{ sec.count }} 部</span>
      </div>

      <p v-if="!sec.rows.length" class="text-default-400 py-3 text-sm">
        本月暂无发售的 Galgame
      </p>

      <div
        v-for="(r, i) in sec.rows"
        :id="`cal-day-${r.id}`"
        :key="r.id"
        class="flex scroll-mt-4 flex-col gap-3 py-4 sm:flex-row sm:gap-5"
        :class="{ 'border-default-200 border-t': i > 0 }"
      >
        <div
          class="flex shrink-0 items-center gap-3 sm:w-16 sm:flex-col sm:gap-1.5"
        >
          <!-- Month above the day number so the date stays unambiguous while
               the section header is scrolled out of view. -->
          <div
            class="flex size-14 flex-col items-center justify-center gap-0.5 rounded-xl"
            :class="tileClass(r)"
          >
            <span class="text-[10px] leading-none opacity-80">
              {{ r.monthShort }}
            </span>
            <KunIcon
              v-if="r.isBucket"
              name="lucide:calendar-clock"
              class="size-5"
            />
            <span v-else class="text-2xl font-bold leading-none">
              {{ r.day }}
            </span>
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
          <!-- 3 columns max: the calendar's 2/3 panel is narrower than the
               full-width lists, so 4-up cards look cramped. -->
          <GalgameCard :galgames="r.games" :columns="3" />
        </div>
      </div>
    </section>
  </div>
</template>
