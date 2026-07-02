<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type { UpdateGalgameSeriesPayload } from '../types'

// Page lives in the URL (?page=N) so the list is shareable / survives
// refresh + back-forward. Default 1 is omitted from the URL. limit is fixed.
const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const limit = 12

const { data, status } = await useKunFetch<{
  series: GalgameSeries[]
  total: number
}>('/galgame-series', {
  method: 'GET',
  query: { page, limit }
})

const showSeriesModal = ref(false)

const handleCreateSeries = async (data: UpdateGalgameSeriesPayload) => {
  // Wiki POST /series expects snake_case keys (docs 04-taxonomy.md).
  // The kungal proxy passes the body verbatim, so the translation
  // has to happen here — submitting `galgame_ids` would land at wiki
  // as an unknown field, the validated `galgame_ids` array stays
  // empty, and wiki returns code:10 / "操作失败".
  const result = await kunFetch(`/galgame-series`, {
    method: 'POST',
    body: {
      name: data.name,
      description: data.description,
      galgame_ids: data.galgame_ids
    }
  })

  if (result) {
    useMessage('创建 Galgame 系列成功', 'success')
  }
}
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="Galgame 系列"
      description="Galgame 全系列所有 Galgame 作品。例如美少女万华镜 1, 2, 3, 4, 5, 雪女, 外传 就是一个 Galgame 系列。某个会社制作的所有 Galgame 并不算系列, 请到 Galgame 会社页面中查看"
    >
      <template #endContent>
        <div class="flex justify-end">
          <KunButton @click="showSeriesModal = true">创建系列</KunButton>
        </div>
      </template>
    </KunHeader>

    <GalgameSeriesModal
      v-model="showSeriesModal"
      :initial-data="{} as UpdateGalgameSeriesPayload"
      @submit="handleCreateSeries"
    />

    <div v-if="data" class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <GalgameSeriesCard
        v-for="(series, index) in data.series"
        :key="series.id"
        :style="{ animationDelay: `${index * 50}ms` }"
        :series="series"
      />
    </div>

    <KunPagination
      v-if="data && data.total > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.total / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
