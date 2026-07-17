<script setup lang="ts">
// Tag management tab: paginated list + debounced search + create / edit /
// two-stage safe delete (mirrors galgame/tag/DetailContainer's flows).
import { watchDebounced } from '@vueuse/core'
import { KUN_GALGAME_TAG_CATEGORY_MAP } from '~/constants/galgameTag'
import type { UpdateGalgameTagPayload } from '~/components/galgame/types'

const { canModerate } = useRole()

const pageData = reactive({ page: 1, limit: 50 })

const { data, status, refresh } = await useKunFetch<{
  tags: GalgameTagItem[]
  total: number
}>(`/galgame-tag`, { method: 'GET', query: pageData })

const searchQuery = ref('')
const searchResult = ref<GalgameTagItem[]>([])
const isSearching = ref(false)

const handleSearch = async () => {
  if (!searchQuery.value.trim()) {
    searchResult.value = []
    return
  }
  isSearching.value = true
  const res = await kunFetch<GalgameTagItem[]>(`/galgame-tag/search`, {
    method: 'GET',
    query: { q: searchQuery.value }
  })
  isSearching.value = false
  searchResult.value = res ?? []
}

watchDebounced(() => searchQuery.value, handleSearch, {
  debounce: 500,
  maxWait: 1000
})

const displayTags = computed(() =>
  searchQuery.value.trim() ? searchResult.value : (data.value?.tags ?? [])
)

const showCreateModal = ref(false)
const showEditModal = ref(false)
const editingTag = ref<UpdateGalgameTagPayload>({} as UpdateGalgameTagPayload)

// The list rows have no description/alias — hydrate the edit modal from the
// detail read (tag_id resolves the entity; page/limit only size the embedded
// galgame list we ignore).
const openEdit = async (tag: GalgameTagItem) => {
  const res = await kunFetch<GalgameTagDetail>(`/galgame-tag/${tag.id}`, {
    method: 'GET',
    query: { page: 1, limit: 1, tag_id: tag.id }
  })
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
  const result = await kunFetch(`/galgame-tag`, { method: 'PUT', body: payload })
  if (result) {
    useMessage('标签已更新', 'success')
    await refresh()
    await handleSearch()
  }
}

// Two-stage safe delete (docs 04-taxonomy): a plain DELETE is rejected while
// the tag is still referenced (wiki toasts the count); only then offer the
// force path that purges relations + hard deletes.
const deletingID = ref(0)
const handleDelete = async (tag: GalgameTagItem) => {
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
    await refresh()
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
        placeholder="输入将会自动搜索标签"
        class-name="flex-1"
      />
      <KunButton @click="showCreateModal = true">新建标签</KunButton>
    </div>

    <p class="text-default-500 text-sm">
      {{ `共 ${data?.total ?? 0} 个标签` }}
    </p>

    <div class="flex flex-col gap-2">
      <div
        v-for="tag in displayTags"
        :key="tag.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <KunLink
            :to="`/galgame-tag/${tag.id}`"
            class-name="truncate font-medium"
          >
            {{ tag.name }}
          </KunLink>
          <KunChip
            size="sm"
            :color="
              tag.category === 'content'
                ? 'primary'
                : tag.category === 'sexual'
                  ? 'danger'
                  : 'success'
            "
          >
            {{ KUN_GALGAME_TAG_CATEGORY_MAP[tag.category] }}
          </KunChip>
          <span class="text-default-500 shrink-0 text-sm">
            {{ tag.galgame_count }} 部
          </span>
        </div>
        <div v-if="canModerate" class="flex shrink-0 gap-2">
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
    <KunNull v-if="!isSearching && !displayTags.length" />

    <KunPagination
      v-if="!searchQuery.trim() && data && data.total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <AdminTaxonomyCreateModal
      v-model="showCreateModal"
      type="tag"
      @created="refresh"
    />
    <GalgameTagModal
      v-model="showEditModal"
      :initial-data="editingTag"
      @submit="handleUpdate"
    />
  </div>
</template>
