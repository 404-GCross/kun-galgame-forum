<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type { UpdateGalgameSeriesPayload } from '../types'

const props = defineProps<{
  data: GalgameSeriesDetail
}>()

// URL-backed page (shareable / back-forward), consistent with the
// tag/official/engine detail pages. The BE returns the full member list
// (a series caps at ~27 games), so we paginate client-side — no extra
// request, just slice the already-loaded cards for display.
const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const limit = 24
const pagedGalgames = computed(() =>
  props.data.galgame.slice((page.value - 1) * limit, page.value * limit)
)

const { canAdminister } = useRole()
const showSeriesModal = ref(false)
const editingSeries = ref<UpdateGalgameSeriesPayload>(
  {} as UpdateGalgameSeriesPayload
)

const openEditSeriesModal = () => {
  if (!props.data) {
    return
  }
  const res = props.data
  editingSeries.value = {
    series_id: res.id,
    name: res.name,
    description: res.description,
    galgame_ids: res.galgame.map((g) => g.id)
  } satisfies UpdateGalgameSeriesPayload
  showSeriesModal.value = true
}

const handleUpdateSeries = async (data: UpdateGalgameSeriesPayload) => {
  // Wiki PUT /series/:id expects snake_case `galgame_ids` (docs
  // 04-taxonomy.md). Same rationale as the create flow in
  // series/Container.vue — kungal proxy passes the body verbatim, so
  // the camelCase → snake_case translation lives at the call site.
  const result = await kunFetch(`/galgame-series/${props.data.id}`, {
    method: 'PUT',
    body: {
      name: data.name,
      description: data.description,
      galgame_ids: data.galgame_ids
    }
  })

  if (result) {
    useMessage('重新编辑成功', 'success')
  }
}

const handleDeleteSeries = async () => {
  const res = await useComponentMessageStore().alert(
    '确定删除这个 Galgame 系列吗?',
    '注意, 删除操作不可撤销'
  )
  if (!res) {
    return
  }

  const result = await kunFetch(`/galgame-series/${props.data.id}`, {
    method: 'DELETE',
    query: { series_id: props.data.id }
  })

  if (result) {
    useMessage('删除 Galgame 系列成功', 'success')
    navigateTo('/galgame-series')
  }
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-3"
  >
    <KunHeader :name="`${data.name} 系列`" :description="data.description">
      <template #endContent>
        <div class="flex flex-col flex-wrap gap-3 text-sm">
          <div class="text-default-500 flex items-center gap-2">
            <KunIcon name="lucide:gamepad-2" />
            <span class="font-medium">
              共计 {{ data.galgame.length }} 部 Galgame
            </span>
          </div>

          <div class="text-default-500 space-x-2">
            <span>
              创建于
              <KunTime :time="data.created" type="date" show-year />
            </span>
            ·
            <span>
              更新于
              <KunTime :time="data.updated" />
            </span>
          </div>

          <div class="flex flex-wrap justify-end gap-1">
            <GalgameRevisionModal
              entity="series"
              :id="data.id"
              :entity-label="`系列「${data.name}」`"
              :can-revert="canAdminister"
            />
            <KunButton
              v-if="canAdminister"
              variant="light"
              color="danger"
              @click="handleDeleteSeries"
            >
              删除系列
            </KunButton>
            <KunButton @click="openEditSeriesModal">编辑系列</KunButton>
          </div>
        </div>
      </template>
    </KunHeader>

    <GalgameSeriesModal
      v-model="showSeriesModal"
      :initial-data="editingSeries"
      @submit="handleUpdateSeries"
    />

    <GalgameCard :is-transparent="false" :galgames="pagedGalgames" />

    <KunPagination
      v-if="data.galgame.length > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.galgame.length / limit)"
    />
  </KunCard>
</template>
