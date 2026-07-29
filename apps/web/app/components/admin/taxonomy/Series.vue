<script setup lang="ts">
// Series management tab: search → create / edit (the shared GalgameSeriesModal
// with its galgame membership picker) / single-step delete (series deletion
// only unbinds members — no force stage).
//
// The PUBLIC series pages retired with the wiki series vocabulary (P3): 146
// wiki series, only 6 of which correspond to anything in the catalog, so there
// was nothing to re-anchor them to. The wiki rows themselves live on until the
// editing engine retires that face, so this staff tab stays — search-driven off
// the staff read-back, wiki ids end to end (doc 106 R11), with no public page
// to link to any more.
import { watchDebounced } from '@vueuse/core'
import type { UpdateGalgameSeriesPayload } from '~/components/galgame/types'

// Proxy-face: taxonomy (tag/official/engine/series) CRUD mirrors infra
// galgame.taxonomy.* (owned by the galgame wiki, not pkg/perm) — stays on
// useRole/canAdminister, not useCan.
const { canAdminister } = useRole()

interface StaffTaxonomyRow {
  id: number
  name: string
}

const searchQuery = ref('')
const searchResult = ref<StaffTaxonomyRow[]>([])
const isSearching = ref(false)

const handleSearch = async () => {
  if (!searchQuery.value.trim()) {
    searchResult.value = []
    return
  }
  isSearching.value = true
  const res = await kunFetch<StaffTaxonomyRow[]>(
    `/galgame-taxonomy/series/search`,
    { method: 'GET', query: { q: searchQuery.value } }
  )
  isSearching.value = false
  searchResult.value = res ?? []
}

watchDebounced(() => searchQuery.value, handleSearch, {
  debounce: 500,
  maxWait: 1000
})

const showSeriesModal = ref(false)
const editingSeries = ref<UpdateGalgameSeriesPayload | undefined>(undefined)

const openCreate = () => {
  editingSeries.value = undefined
  showSeriesModal.value = true
}

// The read-back carries the MEMBERSHIP, which matters more here than anywhere
// else: the update op replaces it wholesale, so an edit that could not read it
// back would empty the series.
const openEdit = async (series: StaffTaxonomyRow) => {
  const res = await kunFetch<UpdateGalgameSeriesPayload & { id: number }>(
    `/galgame-taxonomy/series/${series.id}`,
    { method: 'GET' }
  )
  if (!res) return
  editingSeries.value = {
    series_id: res.id,
    name: res.name,
    description: res.description,
    galgame_ids: res.galgame_ids
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
    await handleSearch()
  }
}

const deletingID = ref(0)
const handleDelete = async (series: StaffTaxonomyRow) => {
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
    await handleSearch()
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center gap-2">
      <KunInput
        v-model="searchQuery"
        type="text"
        placeholder="输入将会自动搜索系列"
        class-name="flex-1"
      />
      <KunButton v-if="canAdminister" @click="openCreate">创建系列</KunButton>
    </div>

    <div class="flex flex-col gap-2">
      <div
        v-for="series in searchResult"
        :key="series.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <!-- No public page to link to: the series browse pages retired
               with the wiki series vocabulary (P3). -->
          <span class="truncate font-medium">{{ series.name }}</span>
        </div>
        <div v-if="canAdminister" class="flex shrink-0 gap-2">
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

    <KunLoading v-if="isSearching" />
    <KunNull
      v-if="!isSearching && !searchResult.length"
      :description="
        searchQuery.trim() ? '未找到匹配的系列' : '搜索系列名以编辑或删除'
      "
    />

    <GalgameSeriesModal
      v-model="showSeriesModal"
      :initial-data="editingSeries"
      @submit="handleSubmit"
    />
  </div>
</template>
