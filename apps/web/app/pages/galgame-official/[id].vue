<script setup lang="ts">
import type { UpdateGalgameOfficialPayload } from '~/components/galgame/types'
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  type KUN_GALGAME_OFFICIAL_TYPE
} from '~/constants/galgameOfficial'

const { canAdminister } = useRole()
const route = useRoute()
const official_id = computed(() => {
  return Number((route.params as { id: string }).id)
})

const { page, limit, type, language, platform, gameType, sortField, sortOrder } =
  useGalgameFilters()

const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
// SFW mode mirrors the server's IsSFW (cookie showKUNGalgameContentLimit !==
// 'nsfw'): the wiki then hides this company's NSFW games from BOTH the list and
// the (content-aware) count, so a NSFW-heavy company can look emptier than it is.
const isSfwMode = computed(() => showKUNGalgameContentLimit.value !== 'nsfw')

const showOfficialModal = ref(false)
const editingOfficial = ref<UpdateGalgameOfficialPayload>(
  {} as UpdateGalgameOfficialPayload
)

// "未发布的游戏": this company's unclaimed VNDB drafts (status=2), scoped by
// official_id. Public claim funnel — open to everyone, not just moderators.
const showDraftsModal = ref(false)

const { data, status } = await useKunFetch<GalgameOfficialDetail>(
  `/galgame-official/${official_id.value}`,
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
      official_id
    }
  }
)

const openEditOfficialModal = () => {
  if (!data.value) {
    return
  }
  const res = data.value
  editingOfficial.value = {
    name: res.name,
    // K-PR6: wiki PR4 sub-change added `Original *string` to
    // UpdateOfficialRequest. Hydrate from detail; empty string when
    // wiki returns no original-language name yet.
    original: res.original ?? '',
    official_id: res.id,
    link: res.link,
    lang: res.lang,
    description: res.description,
    category: res.category as (typeof KUN_GALGAME_OFFICIAL_TYPE)[number],
    alias: res.alias
  } satisfies UpdateGalgameOfficialPayload
  showOfficialModal.value = true
}

const handleUpdateOfficial = async (data: UpdateGalgameOfficialPayload) => {
  const result = await kunFetch(`/galgame-official`, {
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
// + hard delete. admin/moderator only — wiki gates; UI canAdminister (§15.2).
const isDeleting = ref(false)
const handleDeleteOfficial = async () => {
  const ok = await useComponentMessageStore().alert(
    `确定删除会社「${data.value?.name}」吗?`,
    '若该会社未被任何 Galgame 引用将直接删除; 仍被引用时会先提示。'
  )
  if (!ok) return
  isDeleting.value = true
  const res = await kunFetch(`/galgame-official/${official_id.value}`, {
    method: 'DELETE'
  })
  if (res !== null) {
    isDeleting.value = false
    useMessage('会社已删除', 'success')
    await navigateTo('/galgame-official')
    return
  }
  isDeleting.value = false
  const force = await useComponentMessageStore().alert(
    '该会社仍被 Galgame 引用, 删除已被拒绝',
    '强制删除会先清除该会社在所有 Galgame 上的关联, 再硬删除该会社, 不可撤销。确定强制删除吗?'
  )
  if (!force) return
  isDeleting.value = true
  const forced = await kunFetch(`/galgame-official/${official_id.value}`, {
    method: 'DELETE',
    query: { force: true }
  })
  isDeleting.value = false
  if (forced !== null) {
    useMessage('会社已强制删除', 'success')
    await navigateTo('/galgame-official')
  }
}

if (data.value) {
  useKunSeoMeta({
    title: `${data.value.name} 会社`,
    description: `${data.value.name}${data.value.alias?.length ? `, 即 ${data.value.alias.join('| ')}` : ''}, 查看会社 ${data.value.name} 制作的所有 Galgame`
  })
} else {
  useKunDisableSeo('未找到 Galgame 会社')
}
</script>

<template>
  <div v-if="data" class="space-y-6">
    <KunHeader
      :name="`${data.name} 制作的 Galgame`"
      :description="data.description"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            本页展示本站已收录的、由该会社制作的 Galgame,
            可按类型 / 语言 / 平台 / 排序筛选。本站尚未收录的作品不在此列。默认仅显示
            SFW 的 Galgame, 查看 NSFW Galgame 请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>

          <div class="text-default-500">
            会社类别
            <KunChip
              :color="
                data.category === 'company'
                  ? 'primary'
                  : data.category === 'individual'
                    ? 'secondary'
                    : 'success'
              "
            >
              {{ KUN_GALGAME_OFFICIAL_CATEGORY_MAP[data.category] }}
            </KunChip>
          </div>
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
            <KunButton
              variant="flat"
              color="default"
              @click="showDraftsModal = true"
            >
              <KunIcon name="lucide:library-big" />
              未发布的游戏
            </KunButton>
            <GalgameRevisionModal
              entity="official"
              :id="official_id"
              :entity-label="`会社「${data.name}」`"
              :can-revert="canAdminister"
            />
            <template v-if="canAdminister">
              <KunButton @click="openEditOfficialModal">编辑会社</KunButton>
              <KunButton
                variant="flat"
                color="danger"
                :loading="isDeleting"
                @click="handleDeleteOfficial"
              >
                删除会社
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
      description="当前为 SFW 模式，该会社含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <GalgameOfficialModal
      v-model="showOfficialModal"
      :initial-data="editingOfficial"
      @submit="handleUpdateOfficial"
    />

    <GalgameDraftsModal
      v-model="showDraftsModal"
      entity-type="official"
      :entity-id="official_id"
      :entity-name="data.name"
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
      :description="`${data.name} 会社下暂无 Galgame`"
    />
  </div>
</template>
