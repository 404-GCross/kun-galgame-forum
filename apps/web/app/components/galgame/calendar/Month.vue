<script setup lang="ts">
const props = defineProps<{
  data: GalgameCalendarMonth
}>()

// Month paging is owned by the parent (it holds the URL-backed `month` ref).
const emit = defineEmits<{
  prev: []
  next: []
  today: []
}>()

const WEEKDAYS = ['日', '一', '二', '三', '四', '五', '六']

const year = computed(() => Number(props.data.month.slice(0, 4)))
const monthNum = computed(() => Number(props.data.month.slice(5, 7)))

const monthLabel = computed(() => `${year.value} 年 ${monthNum.value} 月`)
const isCurrentMonth = computed(
  () => props.data.month === props.data.today.slice(0, 7)
)
// Backward stops at the data floor; forward is unbounded (parent computes it).
const canGoPrev = computed(() => props.data.month > props.data.meta.minMonth)

// Today's day-of-month, or -1 when today isn't in the viewed month.
const todayDay = computed(() =>
  props.data.today.slice(0, 7) === props.data.month
    ? Number(props.data.today.slice(8, 10))
    : -1
)

// Day-precision groups (keyed by day-of-month) + a month-precision bucket.
const dayGames = computed(() => {
  const map = new Map<number, GalgameCard[]>()
  const bucket: GalgameCard[] = []
  for (const game of props.data.items) {
    if (game.releasePrecision === 'month' || !game.releaseDate) {
      bucket.push(game)
      continue
    }
    const day = Number(game.releaseDate.slice(8, 10))
    if (!map.has(day)) {
      map.set(day, [])
    }
    map.get(day)!.push(game)
  }
  return { map, bucket }
})

const countOf = (day: number | null) =>
  day === null ? 0 : (dayGames.value.map.get(day)?.length ?? 0)

// Calendar cells: leading blanks to align the 1st onto its weekday, the days,
// then trailing blanks to complete the final week. `new Date` is fed fixed
// components → timezone-invariant (SSR-safe).
const cells = computed<(number | null)[]>(() => {
  const firstWeekday = new Date(year.value, monthNum.value - 1, 1).getDay()
  const daysInMonth = new Date(year.value, monthNum.value, 0).getDate()
  const out: (number | null)[] = []
  for (let i = 0; i < firstWeekday; i++) {
    out.push(null)
  }
  for (let d = 1; d <= daysInMonth; d++) {
    out.push(d)
  }
  while (out.length % 7 !== 0) {
    out.push(null)
  }
  return out
})

// `selected` highlights the active cell; clicking scrolls the month list (shown
// on every breakpoint) to that day.
const selected = ref<number | 'bucket' | null>(null)

const defaultSelected = computed<number | 'bucket' | null>(() => {
  const days = [...dayGames.value.map.keys()].sort((a, b) => a - b)
  if (days.length) {
    const cmp = props.data.month.localeCompare(props.data.today.slice(0, 7))
    if (cmp < 0) {
      return days[days.length - 1]! // past month → latest day
    }
    if (cmp > 0) {
      return days[0]! // future month → earliest day
    }
    if (dayGames.value.map.has(todayDay.value)) {
      return todayDay.value
    }
    const prior = days.filter((d) => d <= todayDay.value)
    return prior.length ? prior[prior.length - 1]! : days[0]!
  }
  return dayGames.value.bucket.length ? 'bucket' : null
})

watch(() => props.data.month, () => (selected.value = defaultSelected.value), {
  immediate: true
})

const scrollToRow = (key: string) => {
  if (import.meta.client) {
    document
      .getElementById(`cal-day-${key}`)
      ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}
const pickDay = (day: number | null) => {
  if (day === null || !countOf(day)) {
    return
  }
  selected.value = day
  scrollToRow(String(day))
}
const pickBucket = () => {
  if (!dayGames.value.bucket.length) {
    return
  }
  selected.value = 'bucket'
  scrollToRow('bucket')
}
</script>

<template>
  <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:gap-5">
    <!-- LEFT: compact calendar (a third on wide, sticky so it stays put while
         the month list scrolls). Full width on mobile. -->
    <div
      class="flex flex-col gap-3 lg:sticky lg:top-20 lg:w-1/3 lg:shrink-0 lg:self-start"
    >
      <div
        class="border-default-200 flex items-center justify-between gap-2 rounded-xl border px-2 py-1.5"
      >
        <KunButton
          variant="light"
          :is-icon-only="true"
          :disabled="!canGoPrev"
          @click="emit('prev')"
        >
          <KunIcon name="lucide:chevron-left" class="size-5" />
        </KunButton>
        <div class="flex flex-col items-center">
          <span class="font-bold">{{ monthLabel }}</span>
          <span class="text-default-400 text-xs">共 {{ data.meta.count }} 部</span>
        </div>
        <KunButton variant="light" :is-icon-only="true" @click="emit('next')">
          <KunIcon name="lucide:chevron-right" class="size-5" />
        </KunButton>
      </div>

      <div v-if="!isCurrentMonth" class="flex justify-center">
        <KunButton variant="light" size="sm" @click="emit('today')">
          <KunIcon name="lucide:undo-2" class="size-4" />
          回到本月
        </KunButton>
      </div>

      <div class="grid grid-cols-7 gap-1">
        <div
          v-for="w in WEEKDAYS"
          :key="w"
          class="text-default-400 py-1 text-center text-xs"
        >
          {{ w }}
        </div>
      </div>

      <div class="grid grid-cols-7 gap-1">
        <template v-for="(cell, i) in cells" :key="i">
          <div v-if="cell === null" />
          <button
            v-else
            type="button"
            :disabled="!countOf(cell)"
            class="flex min-h-12 flex-col items-center justify-center gap-0.5 rounded-lg border transition-colors sm:min-h-14"
            :class="[
              selected === cell
                ? 'border-primary bg-default-100'
                : countOf(cell)
                  ? 'border-default-200 hover:border-primary cursor-pointer'
                  : 'border-transparent'
            ]"
            @click="pickDay(cell)"
          >
            <span
              class="text-sm"
              :class="
                cell === todayDay
                  ? 'text-primary font-bold'
                  : countOf(cell)
                    ? 'text-default-600'
                    : 'text-default-400'
              "
            >
              {{ cell }}
            </span>
            <span
              v-if="countOf(cell)"
              class="bg-primary rounded-full px-1.5 text-[10px] font-medium text-white"
            >
              {{ countOf(cell) }}
            </span>
          </button>
        </template>
      </div>

      <button
        v-if="dayGames.bucket.length"
        type="button"
        class="flex items-center gap-2 self-start rounded-lg border px-3 py-2 text-sm transition-colors"
        :class="
          selected === 'bucket'
            ? 'border-primary bg-default-100'
            : 'border-default-200 hover:border-primary'
        "
        @click="pickBucket"
      >
        <KunIcon name="lucide:calendar-clock" class="text-default-500 size-4" />
        本月内 · 日期待定
        <span class="bg-primary rounded-full px-1.5 text-xs text-white">
          {{ dayGames.bucket.length }}
        </span>
      </button>
    </div>

    <!-- The whole month, today pinned on top — shown on every breakpoint
         (beside the calendar on wide, below it on mobile). -->
    <div class="min-w-0 lg:w-2/3">
      <GalgameCalendarMonthList :data="data" />
    </div>
  </div>
</template>
