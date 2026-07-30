<script setup lang="ts">
// Tag management tab: search → edit / two-stage safe delete.
//
// SEARCH-DRIVEN since the A2-3 re-anchoring (doc 106 R11). The public browse
// lanes moved to the CATALOG id space while these write ops still address WIKI
// rows, so a row picked off the public list could not be edited — its id names
// a different entity. The staff face therefore serves this console its own
// search + detail pair, wiki ids end to end, and that pair is search-only:
// pick a row, then read the full record by id. There is no browse list to
// paginate here any more, which is also why the rows show identity alone.
import { watchDebounced } from '@vueuse/core'
import type { UpdateGalgameTagPayload } from '~/components/galgame/types'

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
    `/galgame-taxonomy/tag/search`,
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
const editingTag = ref<UpdateGalgameTagPayload>({} as UpdateGalgameTagPayload)

// The read-back mirrors the update payload field for field. That is the whole
// point of it: this form PUTs a wholesale replacement, so any field it cannot
// read back is a field it silently wipes on save (alias used to go that way on
// every edit).
const openEdit = async (tag: StaffTaxonomyRow) => {
  const res = await kunFetch<UpdateGalgameTagPayload & { id: number }>(
    `/galgame-taxonomy/tag/${tag.id}`,
    { method: 'GET' }
  )
  if (!res) return
  editingTag.value = {
    tag_id: res.id,
    name: res.name,
    category: res.category,
    description: res.description,
    alias: res.alias
  } satisfies UpdateGalgameTagPayload
  showEditModal.value = true
}

const handleUpdate = async (payload: UpdateGalgameTagPayload) => {
  const result = await kunFetch(`/galgame-tag`, {
    method: 'PUT',
    body: payload
  })
  if (result) {
    useMessage('标签已更新', 'success')
    await handleSearch()
  }
}

// Two-stage safe delete (docs 04-taxonomy): a plain DELETE is rejected while
// the tag is still referenced (the wiki toasts the count); only then offer the
// force path that purges relations + hard deletes.
const deletingID = ref(0)
const handleDelete = async (tag: StaffTaxonomyRow) => {
  const ok = await useComponentMessageStore().alert(
    `确定删除标签「${tag.name}」吗?`,
    '若该标签未被任何 Galgame 引用将直接删除; 仍被引用时会先提示。'
  )
  if (!ok) return
  deletingID.value = tag.id
  const res = await kunFetch(`/galgame-tag/${tag.id}`, { method: 'DELETE' })
  if (res !== null) {
    deletingID.value = 0
    useMessage('标签已删除', 'success')
    await handleSearch()
    return
  }
  deletingID.value = 0
  const force = await useComponentMessageStore().alert(
    '该标签仍被 Galgame 引用, 删除已被拒绝',
    '强制删除会先清除该标签在所有 Galgame 上的关联, 再硬删除该标签, 不可撤销。确定强制删除吗?'
  )
  if (!force) return
  deletingID.value = tag.id
  const forced = await kunFetch(`/galgame-tag/${tag.id}`, {
    method: 'DELETE',
    query: { force: true }
  })
  deletingID.value = 0
  if (forced !== null) {
    useMessage('标签已强制删除', 'success')
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
        placeholder="输入将会自动搜索标签"
        class-name="flex-1"
      />
      <KunButton v-if="canAdminister" @click="showCreateModal = true">
        新建标签
      </KunButton>
    </div>

    <div class="flex flex-col gap-2">
      <div
        v-for="tag in searchResult"
        :key="tag.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <!-- A WIKI id, so this deliberately stays on the legacy path: it is a
               301 shell that resolves to the canonical /galgame/tag/{catalogId}
               page in one hop, which is what lets the console link out without
               ever knowing the catalog id. Pointing it straight at the new path
               would feed a wiki id to a route that reads catalog ids — the exact
               mix-up the old `/c/` segment existed to prevent. -->
          <KunLink
            :to="`/galgame-tag/${tag.id}`"
            class-name="truncate font-medium"
          >
            {{ tag.name }}
          </KunLink>
        </div>
        <div v-if="canAdminister" class="flex shrink-0 gap-2">
          <KunButton size="sm" variant="flat" @click="openEdit(tag)">
            编辑
          </KunButton>
          <KunButton
            size="sm"
            variant="flat"
            color="danger"
            :loading="deletingID === tag.id"
            @click="handleDelete(tag)"
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
        searchQuery.trim() ? '未找到匹配的标签' : '搜索标签名以编辑或删除'
      "
    />

    <AdminTaxonomyCreateModal
      v-model="showCreateModal"
      type="tag"
      @created="handleSearch"
    />
    <GalgameTagModal
      v-model="showEditModal"
      :initial-data="editingTag"
      @submit="handleUpdate"
    />
  </div>
</template>
