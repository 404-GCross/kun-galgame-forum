<script setup lang="ts">
// The whole facet in one response (~600 rows), like the engine index: the
// catalog's series lane has no search face behind it, so a name filter can
// only run over the full set. It arrives already sorted by id, which is the
// upstream lane's only order.
const { data } = await useKunFetch<GalgameSeriesItem[]>(`/galgame-series`, {
  method: 'GET'
})

const searchQuery = ref('')
// Substring match over the display name — the only field this lane carries.
// A series' alternate spellings live on the record upstream but are not
// projected onto the index row, so searching for one will miss.
const displaySeries = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) {
    return data.value ?? []
  }
  return (data.value ?? []).filter((s) => s.name.toLowerCase().includes(q))
})
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="Galgame 系列资料库"
      description="这里展示了本站可以检索到的 Galgame 系列, 一个系列是同一部作品的续作、外传与重制版的集合, 例如 ゆずソフト 的 サノバウィッチ 系列。点击系列即可查看该系列下本站已收录的所有 Galgame"
    >
      <template #endContent>
        <div>
          <KunInput
            v-model="searchQuery"
            type="text"
            placeholder="搜索系列名称..."
          />

          <div class="text-default-600 mt-4 flex items-center gap-4 text-sm">
            <span v-if="!searchQuery.trim()">
              {{ `总计 ${data?.length || 0} 个系列` }}
            </span>
            <span v-else>
              {{ `搜索结果: ${displaySeries.length} 个系列` }}
            </span>
          </div>
        </div>
      </template>
    </KunHeader>

    <div
      class="grid grid-cols-2 gap-3 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
    >
      <GalgameSeriesCard
        v-for="series in displaySeries"
        :key="series.id"
        :series="series"
      />
    </div>

    <KunNull v-if="!displaySeries.length" />
  </div>
</template>
