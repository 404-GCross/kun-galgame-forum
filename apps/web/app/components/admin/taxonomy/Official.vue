<script setup lang="ts">
// Official (会社) management tab: paginated list + debounced search + create /
// edit / two-stage safe delete — same flows as the tag tab.
import { watchDebounced } from '@vueuse/core'
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP
} from '~/constants/galgameOfficial'
import type { UpdateGalgameOfficialPayload } from '~/components/galgame/types'

// Proxy-face: taxonomy (tag/official/engine/series) CRUD mirrors infra
// galgame.taxonomy.* (owned by the galgame wiki, not pkg/perm) — stays on
// useRole/canAdminister, not useCan.
const { canAdminister } = useRole()

const pageData = reactive({ page: 1, limit: 50 })

const { data, status, refresh } = await useKunFetch<{
  officials: GalgameOfficialItem[]
  total: number
}>(`/galgame-official`, { method: 'GET', query: pageData })

const searchQuery = ref('')
const searchResult = ref<GalgameOfficialItem[]>([])
const isSearching = ref(false)

const handleSearch = async () => {
  if (!searchQuery.value.trim()) {
    searchResult.value = []
    return
  }
  isSearching.value = true
  const res = await kunFetch<GalgameOfficialItem[]>(
    `/galgame-official/search`,
    { method: 'GET', query: { q: searchQuery.value } }
  )
  isSearching.value = false
  searchResult.value = res ?? []
}

watchDebounced(() => searchQuery.value, handleSearch, {
  debounce: 500,
  maxWait: 1000
})

const displayOfficials = computed(() =>
  searchQuery.value.trim() ? searchResult.value : (data.value?.officials ?? [])
)

const showCreateModal = ref(false)
const showEditModal = ref(false)
const editingOfficial = ref<UpdateGalgameOfficialPayload>(
  {} as UpdateGalgameOfficialPayload
)

// Hydrate the edit modal from the detail read — the list rows carry no
// description / original.
const openEdit = async (official: GalgameOfficialItem) => {
  const res = await kunFetch<GalgameOfficialDetail>(
    `/galgame-official/${official.id}`,
    { method: 'GET', query: { page: 1, limit: 1, official_id: official.id } }
  )
  if (!res) return
  editingOfficial.value = {
    official_id: res.id,
    name: res.name,
    original: res.original ?? '',
    link: res.link,
    lang: res.lang,
    category: res.category,
    description: res.description,
    alias: res.alias
  } satisfies UpdateGalgameOfficialPayload
  showEditModal.value = true
}

const handleUpdate = async (payload: UpdateGalgameOfficialPayload) => {
  const result = await kunFetch(`/galgame-official`, {
    method: 'PUT',
    body: payload
  })
  if (result) {
    useMessage('会社已更新', 'success')
    await refresh()
    await handleSearch()
  }
}

const deletingID = ref(0)
const handleDelete = async (official: GalgameOfficialItem) => {
  const ok = await useComponentMessageStore().alert(
    `确定删除会社「${official.name}」吗?`,
    '若该会社未被任何 Galgame 引用将直接删除; 仍被引用时会先提示。'
  )
  if (!ok) return
  deletingID.value = official.id
  const res = await kunFetch(`/galgame-official/${official.id}`, {
    method: 'DELETE'
  })
  if (res !== null) {
    deletingID.value = 0
    useMessage('会社已删除', 'success')
    await refresh()
    await handleSearch()
    return
  }
  deletingID.value = 0
  const force = await useComponentMessageStore().alert(
    '该会社仍被 Galgame 引用, 删除已被拒绝',
    '强制删除会先清除该会社在所有 Galgame 上的关联, 再硬删除该会社, 不可撤销。确定强制删除吗?'
  )
  if (!force) return
  deletingID.value = official.id
  const forced = await kunFetch(`/galgame-official/${official.id}`, {
    method: 'DELETE',
    query: { force: true }
  })
  deletingID.value = 0
  if (forced !== null) {
    useMessage('会社已强制删除', 'success')
    await refresh()
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
        placeholder="输入将会自动搜索会社"
        class-name="flex-1"
      />
      <KunButton v-if="canAdminister" @click="showCreateModal = true">
        新建会社
      </KunButton>
    </div>

    <p class="text-default-500 text-sm">
      {{ `共 ${data?.total ?? 0} 个会社` }}
    </p>

    <div class="flex flex-col gap-2">
      <div
        v-for="official in displayOfficials"
        :key="official.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <KunLink
            :to="`/galgame-official/${official.id}`"
            class-name="truncate font-medium"
          >
            {{ official.name }}
          </KunLink>
          <KunChip size="sm" color="primary">
            {{ KUN_GALGAME_OFFICIAL_CATEGORY_MAP[official.category] }}
          </KunChip>
          <KunChip v-if="official.lang" size="sm" color="secondary">
            {{
              KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[official.lang] ?? official.lang
            }}
          </KunChip>
          <span class="text-default-500 shrink-0 text-sm">
            {{ official.galgame_count }} 部
          </span>
        </div>
        <div v-if="canAdminister" class="flex shrink-0 gap-2">
          <KunButton size="sm" variant="flat" @click="openEdit(official)">
            编辑
          </KunButton>
          <KunButton
            size="sm"
            variant="flat"
            color="danger"
            :loading="deletingID === official.id"
            @click="handleDelete(official)"
          >
            删除
          </KunButton>
        </div>
      </div>
    </div>

    <KunLoading v-if="isSearching" />
    <KunNull v-if="!isSearching && !displayOfficials.length" />

    <KunPagination
      v-if="!searchQuery.trim() && data && data.total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <AdminTaxonomyCreateModal
      v-model="showCreateModal"
      type="official"
      @created="refresh"
    />
    <GalgameOfficialModal
      v-model="showEditModal"
      :initial-data="editingOfficial"
      @submit="handleUpdate"
    />
  </div>
</template>
