<script setup lang="ts">
import type { UpdateGalgameTagPayload } from '~/components/galgame/types'
import {
  KUN_GALGAME_TAG_CATEGORY_MAP,
  type KUN_GALGAME_TAG_TYPE
} from '~/constants/galgameTag'

const { canModerate } = useRole()
const route = useRoute()
const tagId = computed(() => {
  return Number((route.params as { id: string }).id)
})

const { page, limit, type, language, platform, sortField, sortOrder } =
  useGalgameFilters()

const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
// SFW mode mirrors the server's IsSFW (cookie showKUNGalgameContentLimit !==
// 'nsfw'): the wiki then hides NSFW games from BOTH the list and the
// (content-aware) count, so an NSFW-heavy entity can look emptier than it is.
const isSfwMode = computed(() => showKUNGalgameContentLimit.value !== 'nsfw')

const showTagModal = ref(false)
const editingTag = ref<UpdateGalgameTagPayload>({} as UpdateGalgameTagPayload)

const { data, status } = await useKunFetch<GalgameTagDetail>(
  `/galgame-tag/${tagId.value}`,
  {
    method: 'GET',
    query: {
      page,
      limit,
      type,
      language,
      platform,
      sortField,
      sortOrder,
      tagId
    }
  }
)

const openEditTagModal = () => {
  if (!data.value) {
    return
  }
  const res = data.value
  editingTag.value = {
    name: res.name,
    tagId: res.id,
    description: res.description,
    category: res.category as (typeof KUN_GALGAME_TAG_TYPE)[number],
    alias: res.alias
  } satisfies UpdateGalgameTagPayload
  showTagModal.value = true
}

const handleUpdateTag = async (data: UpdateGalgameTagPayload) => {
  const result = await kunFetch(`/galgame-tag`, {
    method: 'PUT',
    body: data
  })

  if (result) {
    useMessage('重新编辑成功', 'success')
  }
}

if (data.value) {
  if (data.value.category !== 'sexual') {
    useKunSeoMeta({
      title: `标签 ${data.value.name} 的 Galgame`,
      description: data.value.galgame
        .map((g) => getPreferredLanguageText(g.name))
        .toString()
    })
  } else {
    useKunDisableSeo(`标签 ${data.value.name} 的 Galgame`)
  }
} else {
  useKunDisableSeo('未找到 Galgame 标签')
}
</script>

<template>
  <div v-if="data" class="space-y-6">
    <KunHeader
      :name="`含有标签 ${data.name} 的 Galgame`"
      :description="data.description"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            本页展示该标签下的全部 Galgame,
            数据来自百科。标注「未收录」的作品本站尚未收录, 暂无下载资源 /
            评分等信息, 欢迎成为第一个上传资源的人。默认仅显示 SFW 的 Galgame,
            查看 NSFW Galgame 请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>

          <div class="text-default-500">
            标签类别
            <KunChip
              :color="
                data.category === 'content'
                  ? 'primary'
                  : data.category === 'sexual'
                    ? 'danger'
                    : 'success'
              "
            >
              {{ KUN_GALGAME_TAG_CATEGORY_MAP[data.category] }}
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
            <GalgameRevisionModal
              entity="tag"
              :id="tagId"
              :entity-label="`标签「${data.name}」`"
              :can-revert="canModerate"
            />
            <KunButton v-if="canModerate" @click="openEditTagModal">
              编辑标签
            </KunButton>
          </div>
        </div>
      </template>
    </KunHeader>

    <GalgameCardNav :show-advanced="false" />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该标签含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <GalgameTagModal
      v-model="showTagModal"
      :initial-data="editingTag"
      @submit="handleUpdateTag"
    />

    <GalgameCard
      :is-transparent="false"
      v-if="data.galgame.length"
      :galgames="data.galgame"
    />

    <KunPagination
      v-if="data.galgameCount > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.galgameCount / limit)"
      :is-loading="status === 'pending'"
    />

    <KunNull
      v-if="!data.galgameCount"
      :description="`${data.name} 标签下暂无 Galgame`"
    />
  </div>
</template>
