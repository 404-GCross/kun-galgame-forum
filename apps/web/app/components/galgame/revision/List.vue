<script setup lang="ts">
// Universal revision-history list (galgame + 4 taxonomy entities). All
// entity-specific routing lives in useRevisionHistory; this component
// is the visual shell.
//
// Permission gate: `canRevert` is calculated by the parent (the creator
// match + capability check are entity-specific — for galgame it's
// `galgame.user.id === currentUser || canModerate`; for taxonomies it's
// just `canModerate` admin/mod). Parent decides, the list just renders the
// button conditionally.
import type { RevisionEntity } from '~/composables/useRevisionHistory'

const props = defineProps<{
  entity: RevisionEntity
  id: number
  // Static label used in confirm prompts ("回滚标签 / 回滚 Galgame …").
  // Kept as a prop instead of inferring from `entity` so parents can
  // localise / phrase as they like.
  entityLabel?: string
  canRevert?: boolean
}>()

const idRef = computed(() => props.id)
const {
  items,
  total,
  status,
  pageData,
  refresh,
  loadDiff,
  diffCache,
  diffLoading,
  revert,
  reverting
} = useRevisionHistory(props.entity, idRef)

const ACTION_LABEL: Record<string, string> = {
  created: '创建',
  updated: '编辑',
  merged: '合并 PR',
  reverted: '回滚',
  declined: '拒绝 PR',
  deleted: '删除',
  approved: '审核通过',
  banned: '封禁',
  unbanned: '解禁',
  edited_pending: '编辑(待审)',
  claimed: '认领'
}

const openRev = ref<number | null>(null)

const toggleDiff = async (rev: number) => {
  if (openRev.value === rev) {
    openRev.value = null
    return
  }
  openRev.value = rev
  await loadDiff(rev)
}

const handleRevert = async (rev: number) => {
  const ok = await useComponentMessageStore().alert(
    `确定回滚到版本 #${rev} 吗?`,
    `回滚会基于该版本创建一条新的版本记录, 不会删除任何历史。${
      props.entityLabel ? ` 操作对象: ${props.entityLabel}` : ''
    }`
  )
  if (!ok) return
  const success = await revert(rev)
  if (success) {
    openRev.value = null
    useMessage('回滚成功', 'success')
  }
}
</script>

<template>
  <div class="flex flex-col space-y-3">
    <KunHeader
      name="版本历史"
      description="该实体的所有更改历史(创建 / 编辑 / 合并 / 回滚等)"
      scale="h3"
    />

    <KunLoading v-if="status === 'pending'" />

    <KunNull v-else-if="!items.length" description="暂无编辑历史记录" />

    <div
      v-for="rev in items"
      :key="rev.id"
      class="border-default-200 space-y-2 rounded-lg border p-3"
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2 text-sm">
          <KunAvatar :user="rev.user" />
          <div class="space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <span>{{ rev.user.name }}</span>
              <KunChip size="sm">
                {{ ACTION_LABEL[rev.action] || rev.action }}
              </KunChip>
              <span class="text-default-400 text-xs">#{{ rev.revision }}</span>
              <span v-if="rev.isMinor" class="text-default-400 text-xs">
                (小修改)
              </span>
              <span class="text-default-500">
                <KunTime :time="rev.created" />
              </span>
            </div>
            <div class="text-default-500" v-if="rev.note">{{ rev.note }}</div>
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <KunButton
            size="sm"
            variant="flat"
            :loading="diffLoading === rev.revision"
            @click="toggleDiff(rev.revision)"
          >
            {{ openRev === rev.revision ? '收起' : '差异' }}
          </KunButton>
          <KunButton
            v-if="canRevert"
            size="sm"
            variant="flat"
            color="warning"
            :loading="reverting === rev.revision"
            @click="handleRevert(rev.revision)"
          >
            回滚到此版本
          </KunButton>
        </div>
      </div>

      <div
        v-if="openRev === rev.revision && diffCache[rev.revision]"
        class="border-default-200 border-t pt-2"
      >
        <GalgameSnapshotDiff
          :changed-keys="diffCache[rev.revision]!.changed_keys"
          :old-snap="diffCache[rev.revision]!.old"
          :new-snap="diffCache[rev.revision]!.new"
          :names="diffCache[rev.revision]!.names"
        />
      </div>
    </div>

    <KunPagination
      v-if="total >= pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(total / pageData.limit)"
      :is-loading="status === 'pending'"
      @change="refresh"
    />
  </div>
</template>
