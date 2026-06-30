<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type { KunTabItem } from '@kungal/ui-vue'

// URL-backed so a given month / tab is shareable, bookmarkable, and survives
// refresh + back/forward (same idiom as useGalgameFilters). 'month' is the
// default tab and also catches empty / unknown ?view values.
const opts = { mode: 'replace' as const }
// Widen to string (not the literal default) so the union comparisons + the
// KunTab v-model (which emits a bare string) typecheck.
const view = useRouteQuery<string>('view', 'month', opts)
const month = useRouteQuery<string>('month', '', opts) // '' → current month (server-resolved)
const year = useRouteQuery<string>('year', '', opts) // '' → current year (server-resolved)

const tabs: KunTabItem[] = [
  { value: 'month', textValue: '月历' },
  { value: 'pending', textValue: '年内待定' },
  { value: 'tba', textValue: '发售日期未定' }
]

const isMonthView = computed(
  () => view.value !== 'pending' && view.value !== 'tba'
)

// Omit empty month/year so the wiki applies its current-JST default — sending
// `?month=` would trip its strict YYYY-MM validation (→ 400).
const monthQuery = computed(() => (month.value ? { month: month.value } : {}))
const yearQuery = computed(() => (year.value ? { year: year.value } : {}))

// content_limit is derived server-side from the SFW cookie (forwarded on SSR);
// NSFW-opt-in users get the sfw+nsfw union. Each bucket fetches lazily the
// first time its tab opens; the month view is the landing default.
const {
  data: monthData,
  status: monthStatus,
  refresh: refreshMonth
} = await useKunFetch<GalgameCalendarMonth>('/galgame/calendar', {
  method: 'GET',
  query: monthQuery,
  watch: [month],
  immediate: isMonthView.value,
  server: isMonthView.value
})

const {
  data: pendingData,
  status: pendingStatus,
  refresh: refreshPending
} = await useKunFetch<GalgameCalendarPending>('/galgame/calendar/pending', {
  method: 'GET',
  query: yearQuery,
  watch: [year],
  immediate: view.value === 'pending',
  server: view.value === 'pending'
})

const {
  data: tbaData,
  status: tbaStatus,
  refresh: refreshTba
} = await useKunFetch<GalgameCalendarTBA>('/galgame/calendar/tba', {
  method: 'GET',
  immediate: view.value === 'tba',
  server: view.value === 'tba'
})

// Fetch a bucket the first time its tab is activated (lazy tabs start empty).
watch(view, () => {
  if (isMonthView.value && !monthData.value) {
    refreshMonth()
  } else if (view.value === 'pending' && !pendingData.value) {
    refreshPending()
  } else if (view.value === 'tba' && !tbaData.value) {
    refreshTba()
  }
})

const monthLabel = computed(() => {
  const m = monthData.value?.month
  if (!m) {
    return ''
  }
  const [y, mo] = m.split('-')
  return `${y} 年 ${Number(mo)} 月`
})

const goPrevMonth = () => {
  if (monthData.value?.meta.hasPrev) {
    month.value = monthData.value.meta.prevMonth
  }
}
const goNextMonth = () => {
  if (monthData.value?.meta.hasNext) {
    month.value = monthData.value.meta.nextMonth
  }
}
const goToday = () => {
  month.value = ''
}

// Pending exposes no data-boundary meta (just a count), so year nav is
// unbounded — an empty year simply renders the KunNull empty state.
const goPrevYear = () => {
  const y = Number(pendingData.value?.year)
  if (y) {
    year.value = String(y - 1)
  }
}
const goNextYear = () => {
  const y = Number(pendingData.value?.year)
  if (y) {
    year.value = String(y + 1)
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <KunHeader
      name="Galgame 发售月历"
      description="按发售月份浏览即将与已发售的 Galgame。数据来自 Galgame Wiki, 月份边界按日本时间 (JST) 计; 标记「未收录」的作品本站暂无本地数据。"
    />

    <KunTab :items="tabs" v-model="view" />

    <!-- 年内待定 (release_precision=year) -->
    <template v-if="view === 'pending'">
      <div class="flex items-center justify-center gap-3">
        <KunButton variant="light" :is-icon-only="true" @click="goPrevYear">
          <KunIcon name="lucide:chevron-left" />
        </KunButton>
        <span class="text-lg font-medium">
          {{ pendingData?.year }} 年内待定
        </span>
        <KunButton variant="light" :is-icon-only="true" @click="goNextYear">
          <KunIcon name="lucide:chevron-right" />
        </KunButton>
      </div>

      <KunLoading :loading="pendingStatus === 'pending'">
        <template v-if="pendingData">
          <GalgameCard
            v-if="pendingData.items.length"
            :galgames="pendingData.items"
          />
          <KunNull v-else description="该年暂无仅知年份的待定作品" />
        </template>
      </KunLoading>
    </template>

    <!-- 发售日期未定 (release_precision=tba) -->
    <template v-else-if="view === 'tba'">
      <KunLoading :loading="tbaStatus === 'pending'">
        <template v-if="tbaData">
          <GalgameCard v-if="tbaData.items.length" :galgames="tbaData.items" />
          <KunNull v-else description="暂无发售日期未定的作品" />
        </template>
      </KunLoading>
    </template>

    <!-- 月历 (default) -->
    <template v-else>
      <div class="flex items-center justify-center gap-2">
        <KunButton
          variant="light"
          :is-icon-only="true"
          :disabled="!monthData?.meta.hasPrev"
          @click="goPrevMonth"
        >
          <KunIcon name="lucide:chevron-left" />
        </KunButton>
        <span class="min-w-32 text-center text-lg font-medium">
          {{ monthLabel }}
        </span>
        <KunButton
          variant="light"
          :is-icon-only="true"
          :disabled="!monthData?.meta.hasNext"
          @click="goNextMonth"
        >
          <KunIcon name="lucide:chevron-right" />
        </KunButton>
        <KunButton variant="flat" size="sm" @click="goToday">
          回到本月
        </KunButton>
      </div>

      <KunLoading :loading="monthStatus === 'pending'">
        <GalgameCalendarMonth
          v-if="monthData && monthData.items.length"
          :data="monthData"
        />
        <KunNull v-else-if="monthData" description="本月暂无发售的 Galgame" />
      </KunLoading>
    </template>
  </div>
</template>
