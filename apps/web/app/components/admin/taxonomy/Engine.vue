<script setup lang="ts">
// Engine management tab: search → edit / two-stage safe delete.
//
// SEARCH-DRIVEN since the A2-3 re-anchoring — see the tag tab for why the
// public browse lanes can no longer feed a wiki-id edit form (doc 106 R11).
import { watchDebounced } from '@vueuse/core'
import type { UpdateGalgameEnginePayload } from '~/components/galgame/types'

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
    `/galgame-taxonomy/engine/search`,
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
const editingEngine = ref<UpdateGalgameEnginePayload>(
  {} as UpdateGalgameEnginePayload
)

// The read-back mirrors the update payload field for field — this form PUTs a
// wholesale replacement, so any field it cannot read back is a field it wipes
// on save.
const openEdit = async (engine: StaffTaxonomyRow) => {
  const res = await kunFetch<UpdateGalgameEnginePayload & { id: number }>(
    `/galgame-taxonomy/engine/${engine.id}`,
    { method: 'GET' }
  )
  if (!res) return
  editingEngine.value = {
    engine_id: res.id,
    name: res.name,
    description: res.description ?? '',
    alias: res.alias
  } satisfies UpdateGalgameEnginePayload
  showEditModal.value = true
}

const handleUpdate = async (payload: UpdateGalgameEnginePayload) => {
  const result = await kunFetch(`/galgame-engine`, {
    method: 'PUT',
    body: payload
  })
  if (result) {
    useMessage('引擎已更新', 'success')
    await handleSearch()
  }
}

const deletingID = ref(0)
const handleDelete = async (engine: StaffTaxonomyRow) => {
  const ok = await useComponentMessageStore().alert(
    `确定删除引擎「${engine.name}」吗?`,
    '若该引擎未被任何 Galgame 引用将直接删除; 仍被引用时会先提示。'
  )
  if (!ok) return
  deletingID.value = engine.id
  const res = await kunFetch(`/galgame-engine/${engine.id}`, {
    method: 'DELETE'
  })
  if (res !== null) {
    deletingID.value = 0
    useMessage('引擎已删除', 'success')
    await handleSearch()
    return
  }
  deletingID.value = 0
  const force = await useComponentMessageStore().alert(
    '该引擎仍被 Galgame 引用, 删除已被拒绝',
    '强制删除会先清除该引擎在所有 Galgame 上的关联, 再硬删除该引擎, 不可撤销。确定强制删除吗?'
  )
  if (!force) return
  deletingID.value = engine.id
  const forced = await kunFetch(`/galgame-engine/${engine.id}`, {
    method: 'DELETE',
    query: { force: true }
  })
  deletingID.value = 0
  if (forced !== null) {
    useMessage('引擎已强制删除', 'success')
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
        placeholder="输入将会自动搜索引擎"
        class-name="flex-1"
      />
      <KunButton v-if="canAdminister" @click="showCreateModal = true">
        新建引擎
      </KunButton>
    </div>

    <div class="flex flex-col gap-2">
      <div
        v-for="engine in searchResult"
        :key="engine.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <!-- A WIKI id, so this deliberately stays on the legacy path: it is a
               301 shell that resolves to the canonical /galgame/engine/{catalogId}
               page in one hop. Pointing it straight at the new path would feed a
               wiki id to a route that reads catalog ids — the exact mix-up the
               old `/c/` segment existed to prevent. -->
          <KunLink
            :to="`/galgame-engine/${engine.id}`"
            class-name="truncate font-medium"
          >
            {{ engine.name }}
          </KunLink>
        </div>
        <div v-if="canAdminister" class="flex shrink-0 gap-2">
          <KunButton size="sm" variant="flat" @click="openEdit(engine)">
            编辑
          </KunButton>
          <KunButton
            size="sm"
            variant="flat"
            color="danger"
            :loading="deletingID === engine.id"
            @click="handleDelete(engine)"
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
        searchQuery.trim() ? '未找到匹配的引擎' : '搜索引擎名以编辑或删除'
      "
    />

    <AdminTaxonomyCreateModal
      v-model="showCreateModal"
      type="engine"
      @created="handleSearch"
    />
    <GalgameEngineModal
      v-model="showEditModal"
      :initial-data="editingEngine"
      @submit="handleUpdate"
    />
  </div>
</template>
