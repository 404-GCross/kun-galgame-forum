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

const pageData = reactive({
  page: 1,
  limit: 24,
  tagId: tagId.value
})

const showTagModal = ref(false)
const editingTag = ref<UpdateGalgameTagPayload>({} as UpdateGalgameTagPayload)

const { data, status } = await useKunFetch<GalgameTagDetail>(
  `/galgame-tag/${tagId.value}`,
  {
    method: 'GET',
    query: pageData
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

// Two-stage safe delete (docs 04-taxonomy / 00-handbook): a plain
// DELETE /galgame-tag/:id is REJECTED while the tag is still referenced
// by any galgame (wiki returns the reference count, surfaced by
// kunFetch as a toast). Only then, after an explicit second
// confirmation, retry with ?force=true to purge all relations + hard
// delete. admin/moderator only — wiki gates; UI gated canModerate (§15.2).
const isDeleting = ref(false)
const handleDeleteTag = async () => {
  const ok = await useComponentMessageStore().alert(
    `确定删除标签「${data.value?.name}」吗?`,
    '若该标签未被任何 Galgame 引用将直接删除; 仍被引用时会先提示。'
  )
  if (!ok) return
  isDeleting.value = true
  const res = await kunFetch(`/galgame-tag/${tagId.value}`, {
    method: 'DELETE'
  })
  if (res !== null) {
    isDeleting.value = false
    useMessage('标签已删除', 'success')
    await navigateTo('/galgame-tag')
    return
  }
  // Rejected — almost always "still referenced" (wiki already toasted
  // the count). Offer the force path explicitly.
  isDeleting.value = false
  const force = await useComponentMessageStore().alert(
    '该标签仍被 Galgame 引用, 删除已被拒绝',
    '强制删除会先清除该标签在所有 Galgame 上的关联, 再硬删除该标签, 不可撤销。确定强制删除吗?'
  )
  if (!force) return
  isDeleting.value = true
  const forced = await kunFetch(`/galgame-tag/${tagId.value}`, {
    method: 'DELETE',
    query: { force: true }
  })
  isDeleting.value = false
  if (forced !== null) {
    useMessage('标签已强制删除', 'success')
    await navigateTo('/galgame-tag')
  }
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
            默认仅显示了 SFW 的 Galgame, 查看 NSFW Galgame 请在设置面板打开 NSFW
            开关。如果有数据错误请
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
          <div v-if="canModerate" class="flex justify-end gap-2">
            <KunButton @click="openEditTagModal">编辑标签</KunButton>
            <KunButton
              variant="flat"
              color="danger"
              :loading="isDeleting"
              @click="handleDeleteTag"
            >
              删除标签
            </KunButton>
          </div>
        </div>
      </template>
    </KunHeader>

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
      v-if="data.galgameCount > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.galgameCount / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <KunNull
      v-if="!data.galgameCount"
      :description="`${data.name} 标签下暂无 Galgame`"
    />

    <GalgameRevisionList
      entity="tag"
      :id="tagId"
      :entity-label="`标签「${data.name}」`"
      :can-revert="canModerate"
    />
  </div>
</template>
