<script setup lang="ts">
// Series management tab: paginated list + create / edit (the shared
// GalgameSeriesModal with its galgame membership picker) / single-step
// delete (series deletion only unbinds members — no force stage).
import type { UpdateGalgameSeriesPayload } from '~/components/galgame/types'

const { canModerate } = useRole()

const pageData = reactive({ page: 1, limit: 24 })

const { data, status, refresh } = await useKunFetch<{
  series: GalgameSeries[]
  total: number
}>(`/galgame-series`, { method: 'GET', query: pageData })

const showSeriesModal = ref(false)
const editingSeries = ref<UpdateGalgameSeriesPayload | undefined>(undefined)

const openCreate = () => {
  editingSeries.value = undefined
  showSeriesModal.value = true
}

// Hydrate membership ids from the detail read (the list rows only carry
// counts + samples).
const openEdit = async (series: GalgameSeries) => {
  const res = await kunFetch<GalgameSeriesDetail>(
    `/galgame-series/${series.id}`,
    { method: 'GET' }
  )
  if (!res) return
  editingSeries.value = {
    series_id: res.id,
    name: res.name,
    description: res.description,
    galgame_ids: res.galgame.map((g) => g.id)
  } satisfies UpdateGalgameSeriesPayload
  showSeriesModal.value = true
}

// Wiki POST/PUT expect snake_case keys; the proxy forwards the body verbatim
// (same rationale as galgame/series/Container.vue).
const handleSubmit = async (payload: UpdateGalgameSeriesPayload) => {
  const body = {
    name: payload.name,
    description: payload.description,
    galgame_ids: payload.galgame_ids
  }
  const result = payload.series_id
    ? await kunFetch(`/galgame-series/${payload.series_id}`, {
        method: 'PUT',
        body
      })
    : await kunFetch(`/galgame-series`, { method: 'POST', body })
  if (result) {
    useMessage(payload.series_id ? '系列已更新' : '系列已创建', 'success')
    await refresh()
  }
}

const deletingID = ref(0)
const handleDelete = async (series: GalgameSeries) => {
  const ok = await useComponentMessageStore().alert(
    `确定删除系列「${series.name}」吗?`,
    '删除系列只会解除成员 Galgame 的关联, 不影响 Galgame 本体。此操作不可撤销。'
  )
  if (!ok) return
  deletingID.value = series.id
  const result = await kunFetch(`/galgame-series/${series.id}`, {
    method: 'DELETE',
    query: { series_id: series.id }
  })
  deletingID.value = 0
  if (result) {
    useMessage('系列已删除', 'success')
    await refresh()
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2">
      <p class="text-default-500 text-sm">
        {{ `共 ${data?.total ?? 0} 个系列` }}
      </p>
      <KunButton @click="openCreate">创建系列</KunButton>
    </div>

    <div class="flex flex-col gap-2">
      <div
        v-for="series in data?.series ?? []"
        :key="series.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <KunLink
            :to="`/galgame-series/${series.id}`"
            class-name="truncate font-medium"
          >
            {{ series.name }}
          </KunLink>
          <span class="text-default-500 shrink-0 text-sm">
            {{ series.galgame_count }} 部
          </span>
        </div>
        <div v-if="canModerate" class="flex shrink-0 gap-2">
          <KunButton size="sm" variant="flat" @click="openEdit(series)">
            编辑
          </KunButton>
          <KunButton
            size="sm"
            variant="flat"
            color="danger"
            :loading="deletingID === series.id"
            @click="handleDelete(series)"
          >
            删除
          </KunButton>
        </div>
      </div>
    </div>

    <KunNull v-if="!data?.series?.length" />

    <KunPagination
      v-if="data && data.total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <GalgameSeriesModal
      v-model="showSeriesModal"
      :initial-data="editingSeries"
      @submit="handleSubmit"
    />
  </div>
</template>
