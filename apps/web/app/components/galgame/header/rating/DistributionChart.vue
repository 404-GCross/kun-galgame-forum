<script setup lang="ts">
import VueApexCharts from 'vue3-apexcharts'
import type { ApexOptions } from 'apexcharts'

const props = defineProps<{
  galgameId: number
  source: string
  buckets: number[]
  mine: number | null
  unit?: string
}>()

const colorMode = useColorMode()

const mineIndex = computed(() => (props.mine == null ? -1 : props.mine - 1))

const series = computed(() => [{ name: '评分人数', data: props.buckets }])

const colors = computed(() =>
  props.buckets.map((_, index) =>
    index === mineIndex.value ? 'var(--color-warning)' : 'var(--color-primary)'
  )
)

const options = computed(
  (): ApexOptions => ({
    chart: {
      id: `galgame-rating-distribution-${props.source}-${props.galgameId}`,
      type: 'bar',
      height: 200,
      toolbar: { show: false },
      fontFamily: 'inherit'
    },
    plotOptions: {
      bar: { distributed: true, borderRadius: 4, columnWidth: '55%' }
    },
    dataLabels: { enabled: false },
    colors: colors.value,
    xaxis: {
      categories: props.buckets.map((_, index) => String(index + 1)),
      axisTicks: { show: false },
      labels: { style: { colors: 'var(--color-default-500)' } }
    },
    yaxis: {
      labels: {
        formatter: (value: number) => String(Math.round(value)),
        style: { colors: 'var(--color-default-500)' }
      }
    },
    tooltip: {
      theme: colorMode.preference,
      x: { formatter: (x: number) => `${x} ${props.unit ?? '分'}` },
      y: { formatter: (y: number) => `${y} 人` }
    },
    grid: { borderColor: 'var(--color-default-200)', strokeDashArray: 4 },
    legend: { show: false }
  })
)
</script>

<template>
  <ClientOnly>
    <VueApexCharts
      type="bar"
      height="200"
      :options="options"
      :series="series"
    />
    <template #fallback>
      <div class="h-[200px]" />
    </template>
  </ClientOnly>
</template>
