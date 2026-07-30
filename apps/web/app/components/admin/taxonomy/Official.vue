<script setup lang="ts">
// Official (会社) management tab: search → edit / two-stage safe delete.
//
// SEARCH-DRIVEN since the A2-3 re-anchoring — see the tag tab for why the
// public browse lanes can no longer feed a wiki-id edit form (doc 106 R11).
import { watchDebounced } from '@vueuse/core'
import type { UpdateGalgameOfficialPayload } from '~/components/galgame/types'

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
    `/galgame-taxonomy/official/search`,
    { method: 'GET', query: { q: searchQuery.value } }
  )
  isSearching.value = false
  searchResult.value = res ?? []
}

watchDebounced(() => searchQuery.value, handleSearch, {
  debounce: 500,
  maxWait: 1000
})

const showCreateModal = ref(false)
const showEditModal = ref(false)
const editingOfficial = ref<UpdateGalgameOfficialPayload>(
  {} as UpdateGalgameOfficialPayload
)

// The read-back mirrors the update payload field for field — this form PUTs a
// wholesale replacement, so any field it cannot read back is a field it wipes
// on save (`original` and `alias` both used to go that way).
const openEdit = async (official: StaffTaxonomyRow) => {
  const res = await kunFetch<UpdateGalgameOfficialPayload & { id: number }>(
    `/galgame-taxonomy/official/${official.id}`,
    { method: 'GET' }
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
    await handleSearch()
  }
}

const deletingID = ref(0)
const handleDelete = async (official: StaffTaxonomyRow) => {
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

    <div class="flex flex-col gap-2">
      <div
        v-for="official in searchResult"
        :key="official.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <!-- A WIKI id, so this deliberately stays on the legacy path: it is a
               301 shell that resolves to the canonical /galgame/official/{catalogId}
               page in one hop. Pointing it straight at the new path would feed a
               wiki id to a route that reads catalog ids — the exact mix-up the
               old `/c/` segment existed to prevent. -->
          <KunLink
            :to="`/galgame-official/${official.id}`"
            class-name="truncate font-medium"
          >
            {{ official.name }}
          </KunLink>
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
    <KunNull
      v-if="!isSearching && !searchResult.length"
      :description="
        searchQuery.trim() ? '未找到匹配的会社' : '搜索会社名以编辑或删除'
      "
    />

    <AdminTaxonomyCreateModal
      v-model="showCreateModal"
      type="official"
      @created="handleSearch"
    />
    <GalgameOfficialModal
      v-model="showEditModal"
      :initial-data="editingOfficial"
      @submit="handleUpdate"
    />
  </div>
</template>
