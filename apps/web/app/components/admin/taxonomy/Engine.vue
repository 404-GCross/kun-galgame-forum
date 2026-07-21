<script setup lang="ts">
// Engine management tab: the wiki engine list is flat (no server pagination
// or search endpoint), so filtering is client-side by name/alias — the same
// posture as the public engine list page.
import type { UpdateGalgameEnginePayload } from '~/components/galgame/types'

// Proxy-face: taxonomy (tag/official/engine/series) CRUD mirrors infra
// galgame.taxonomy.* (owned by the galgame wiki, not pkg/perm) — stays on
// useRole/canAdminister, not useCan.
const { canAdminister } = useRole()

const { data, refresh } = await useKunFetch<GalgameEngineItem[]>(
  `/galgame-engine`,
  { method: 'GET' }
)

const searchQuery = ref('')

const displayEngines = computed(() => {
  const engines = data.value ?? []
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return engines
  return engines.filter(
    (e) =>
      e.name.toLowerCase().includes(q) ||
      e.alias.some((a) => a.toLowerCase().includes(q))
  )
})

const showCreateModal = ref(false)
const showEditModal = ref(false)
const editingEngine = ref<UpdateGalgameEnginePayload>(
  {} as UpdateGalgameEnginePayload
)

// The flat list already carries description + alias — no detail hydration.
const openEdit = (engine: GalgameEngineItem) => {
  editingEngine.value = {
    engine_id: engine.id,
    name: engine.name,
    description: engine.description ?? '',
    alias: engine.alias
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
    await refresh()
  }
}

const deletingID = ref(0)
const handleDelete = async (engine: GalgameEngineItem) => {
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
    await refresh()
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
    await refresh()
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center gap-2">
      <KunInput
        v-model="searchQuery"
        type="text"
        placeholder="输入引擎名或别名以筛选"
        class-name="flex-1"
      />
      <KunButton v-if="canAdminister" @click="showCreateModal = true">
        新建引擎
      </KunButton>
    </div>

    <p class="text-default-500 text-sm">
      {{ `共 ${data?.length ?? 0} 个引擎` }}
    </p>

    <div class="flex flex-col gap-2">
      <div
        v-for="engine in displayEngines"
        :key="engine.id"
        class="border-default-200 flex items-center justify-between gap-3 rounded-lg border p-3"
      >
        <div class="flex min-w-0 items-center gap-2">
          <KunLink
            :to="`/galgame-engine/${engine.id}`"
            class-name="truncate font-medium"
          >
            {{ engine.name }}
          </KunLink>
          <span class="text-default-500 shrink-0 text-sm">
            {{ engine.galgame_count }} 部
          </span>
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

    <KunNull v-if="!displayEngines.length" />

    <AdminTaxonomyCreateModal
      v-model="showCreateModal"
      type="engine"
      @created="refresh"
    />
    <GalgameEngineModal
      v-model="showEditModal"
      :initial-data="editingEngine"
      @submit="handleUpdate"
    />
  </div>
</template>
