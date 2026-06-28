<script setup lang="ts">
// PR review: renders the proposed change (GalgameSnapshotDiff: base
// revision snapshot → pr.snapshot, limited to changed_keys) plus
// merge / decline for the creator or admin/moderator. Endpoints verified
// against router.go: PUT /galgame/:gid/prs/:id/merge (298) and
// /decline (301, body {note}). Only pending (status=0) PRs are
// actionable; merged/declined are view-only.
const props = defineProps<{
  details: GalgamePRDiffView
  refresh: () => void
}>()
const galgame = inject<GalgameDetail>('galgame')

const { id } = usePersistUserStore()
const { canModerate } = useRole()
const isShowButton = computed(
  () => galgame?.user.id === id || canModerate.value
)
const isPending = computed(() => props.details.status === 0)
const isFetching = ref(false)
const isShowReasonInput = ref(false)
const declineInput = ref('')

const handleDeclineRequest = async () => {
  if (!declineInput.value.trim() || declineInput.value.trim().length > 1007) {
    useMessage(10543, 'warn')
    return
  }
  const res = await useComponentMessageStore().alert(
    '您确定拒绝更新请求吗？',
    '这将不会将该更新合并至当前的 Galgame 信息中。'
  )
  if (!res) {
    return
  }

  isFetching.value = true
  const result = await kunFetch(
    `/galgame/${props.details.galgameId}/prs/${props.details.id}/decline`,
    {
      method: 'PUT',
      body: { note: declineInput.value.trim() }
    }
  )
  isFetching.value = false

  if (result) {
    useMessage(10544, 'success')
    props.refresh()
  }
}

const handleMergeRequest = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定合并更新请求吗？',
    '这将会立即将更新请求中的内容合并到当前 Galgame 中。'
  )
  if (!res) {
    return
  }

  isFetching.value = true
  const result = await kunFetch(
    `/galgame/${props.details.galgameId}/prs/${props.details.id}/merge`,
    { method: 'PUT' }
  )
  isFetching.value = false

  if (result) {
    useMessage(10545, 'success')
    props.refresh()
  }
}
</script>

<template>
  <div class="border-default-200 space-y-3 rounded-lg border p-3">
    <GalgameSnapshotDiff
      :changed-keys="details.changedKeys"
      :old-snap="details.oldSnap"
      :new-snap="details.newSnap"
      :names="details.names"
    />

    <div
      class="flex items-center justify-end gap-1"
      v-if="isPending && isShowButton"
    >
      <KunButton
        variant="light"
        color="danger"
        @click="isShowReasonInput = !isShowReasonInput"
        :loading="isFetching"
      >
        拒绝
      </KunButton>
      <KunButton @click="handleMergeRequest" :loading="isFetching">
        合并
      </KunButton>
    </div>

    <div
      class="text-default-500 text-sm"
      v-else-if="isPending && !isShowButton"
    >
      要处理该请求, 需要是该资源的发布者或管理员
    </div>

    <div class="flex items-center gap-1" v-if="isShowReasonInput">
      <KunInput placeholder="请填写拒绝更新请求的理由" v-model="declineInput" />
      <KunButton
        color="danger"
        @click="handleDeclineRequest"
        :loading="isFetching"
        class-name="shrink-0"
      >
        确定拒绝
      </KunButton>
    </div>
  </div>
</template>
