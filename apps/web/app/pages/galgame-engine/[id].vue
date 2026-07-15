<script setup lang="ts">
import type { UpdateGalgameEnginePayload } from '~/components/galgame/types'

const { canModerate } = useRole()
const route = useRoute()
const engine_id = computed(() => {
  return Number((route.params as { id: string }).id)
})

// Shared browse filter Nav with /galgame-tag/[id].vue + /galgame-official/[id].vue:
// the entity detail lists the forum-LOCAL subset of the engine's catalogue, so
// the same 类型/语言/平台/作品类型 filters + sorts as /galgame apply (backend runs
// them locally over the engine's member ids — see entity_filter.buildEntityFilter).
const { page, limit, type, language, platform, gameType, sortField, sortOrder } =
  useGalgameFilters()

const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
// SFW mode mirrors the server's IsSFW (cookie showKUNGalgameContentLimit !==
// 'nsfw'): the wiki then hides NSFW games from BOTH the list and the
// (content-aware) count, so an NSFW-heavy entity can look emptier than it is.
const isSfwMode = computed(() => showKUNGalgameContentLimit.value !== 'nsfw')

const showEngineModal = ref(false)
const editingEngine = ref<UpdateGalgameEnginePayload>(
  {} as UpdateGalgameEnginePayload
)

const { data, status } = await useKunFetch<GalgameEngineDetail>(
  `/galgame-engine/${engine_id.value}`,
  {
    method: 'GET',
    query: {
      page,
      limit,
      type,
      language,
      platform,
      gameType,
      sortField,
      sortOrder,
      engine_id
    }
  }
)

const openEditEngineModal = () => {
  if (!data.value) {
    return
  }
  const res = data.value
  editingEngine.value = {
    engine_id: res.id,
    name: res.name,
    description: res.description,
    alias: res.alias
  } satisfies UpdateGalgameEnginePayload
  showEngineModal.value = true
}

const handleUpdateEngine = async (data: UpdateGalgameEnginePayload) => {
  const result = await kunFetch(`/galgame-engine`, {
    method: 'PUT',
    body: data
  })

  if (result) {
    useMessage('重新编辑成功', 'success')
  }
}

// Two-stage safe delete (docs 04-taxonomy / 00-handbook): plain DELETE
// is rejected while still referenced (wiki toasts the count); only after
// an explicit second confirm do we retry ?force=true to purge relations
// + hard delete. admin/moderator only — wiki gates; UI canModerate (§15.2).
const isDeleting = ref(false)
const handleDeleteEngine = async () => {
  const ok = await useComponentMessageStore().alert(
    `确定删除引擎「${data.value?.name}」吗?`,
    '若该引擎未被任何 Galgame 引用将直接删除; 仍被引用时会先提示。'
  )
  if (!ok) return
  isDeleting.value = true
  const res = await kunFetch(`/galgame-engine/${engine_id.value}`, {
    method: 'DELETE'
  })
  if (res !== null) {
    isDeleting.value = false
    useMessage('引擎已删除', 'success')
    await navigateTo('/galgame-engine')
    return
  }
  isDeleting.value = false
  const force = await useComponentMessageStore().alert(
    '该引擎仍被 Galgame 引用, 删除已被拒绝',
    '强制删除会先清除该引擎在所有 Galgame 上的关联, 再硬删除该引擎, 不可撤销。确定强制删除吗?'
  )
  if (!force) return
  isDeleting.value = true
  const forced = await kunFetch(`/galgame-engine/${engine_id.value}`, {
    method: 'DELETE',
    query: { force: true }
  })
  isDeleting.value = false
  if (forced !== null) {
    useMessage('引擎已强制删除', 'success')
    await navigateTo('/galgame-engine')
  }
}

if (data.value) {
  useKunSeoMeta({
    title: `${data.value.name} 引擎`,
    description: `查看所有使用 ${data.value.name} 引擎制作的 Galgame`
  })
} else {
  useKunDisableSeo('未找到 Galgame 引擎')
}
</script>

<template>
  <div v-if="data" class="space-y-6">
    <KunHeader
      :name="`${data.name} 引擎制作的 Galgame`"
      :description="data.description"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            本页展示本站已收录的、使用该引擎制作的 Galgame,
            可按类型 / 语言 / 平台 / 排序筛选。本站尚未收录的作品不在此列。默认仅显示
            SFW 的 Galgame, 查看 NSFW Galgame 请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>

          <div
            v-if="data.alias.length"
            class="text-default-500 flex flex-wrap gap-2"
          >
            别名
            <KunChip
              color="primary"
              v-for="(a, index) in data.alias"
              :key="index"
            >
              {{ a }}
            </KunChip>
          </div>
          <div class="flex flex-wrap justify-end gap-2">
            <GalgameRevisionModal
              entity="engine"
              :id="engine_id"
              :entity-label="`引擎「${data.name}」`"
              :can-revert="canModerate"
            />
            <template v-if="canModerate">
              <KunButton @click="openEditEngineModal">编辑引擎</KunButton>
              <KunButton
                variant="flat"
                color="danger"
                :loading="isDeleting"
                @click="handleDeleteEngine"
              >
                删除引擎
              </KunButton>
            </template>
          </div>
        </div>
      </template>
    </KunHeader>

    <GalgameCardNav :show-advanced="false" />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该引擎含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <GalgameEngineModal
      v-model="showEngineModal"
      :initial-data="editingEngine"
      @submit="handleUpdateEngine"
    />

    <GalgameCard
      :is-transparent="false"
      v-if="data.galgame.length"
      :galgames="data.galgame"
    />

    <KunPagination
      v-if="data.galgame_count > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.galgame_count / limit)"
      :is-loading="status === 'pending'"
    />

    <KunNull
      v-if="!data.galgame_count"
      :description="`${data.name} 引擎下暂无 Galgame`"
    />
  </div>
</template>
