<script setup lang="ts">
const props = defineProps<{
  series: GalgameDetailSeriesRef[]
}>()

const ids = computed(() => props.series.map((s) => s.id).join(','))

const { data } = await useKunFetch<{
  series: GalgameSeriesCard[]
  total: number
}>('/galgame-series/cards', {
  lazy: true,
  method: 'GET',
  query: { ids },
  watch: false
})
</script>

<template>
  <div v-if="data?.series.length" class="space-y-3">
    <KunHeader
      name="Galgame 系列"
      description="Galgame 全系列所有 Galgame 作品。例如美少女万华镜 1, 2, 3, 4, 5, 雪女, 外传 就是一个 Galgame 系列"
      scale="h3"
    />

    <div class="grid grid-cols-1 gap-6">
      <GalgameSeriesCard
        v-for="item in data.series"
        :key="item.id"
        :series="item"
      />
    </div>
  </div>
</template>
