<script setup lang="ts">
import { KUN_GALGAME_RESOURCE_PULL_REQUEST_STATUS_MAP } from '~/constants/galgame'
const props = defineProps<{
  galgameId: number
  pr: GalgamePR
  status: UseFetchStatus
  refresh: () => void
}>()

const iconMap: Record<number, string> = {
  0: 'lucide:loader',
  1: 'lucide:check',
  2: 'lucide:x'
}
const statusColorMap: Record<number, string> = {
  0: 'text-primary',
  1: 'text-success',
  2: 'text-danger'
}

const details = ref<GalgamePRDiffView>()
const isFetching = ref(false)

// Wiki GET /galgame/:gid/prs/:id is passed through verbatim and now
// returns { pr: { snapshot, base_revision, ... }, changed_keys }
// (NOT the old oldData/newData). To show old → new we pair pr.snapshot
// (new) with the base revision's snapshot (old), limited to
// changed_keys. See docs/galgame_wiki/02-revisions-and-prs.md.
const handleGetDetails = async (galgamePrId: number) => {
  if (details.value) {
    return
  }
  isFetching.value = true
  const detail = await kunFetch<WikiPRDetailResponse>(
    `/galgame/${props.galgameId}/prs/${galgamePrId}`,
    { method: 'GET' }
  )
  if (!detail?.pr) {
    isFetching.value = false
    return
  }
  const baseRev = await kunFetch<{ snapshot?: Record<string, unknown> }>(
    `/galgame/${props.galgameId}/revisions/${detail.pr.base_revision}`,
    { method: 'GET' }
  )
  isFetching.value = false

  details.value = {
    id: detail.pr.id,
    galgame_id: detail.pr.galgame_id,
    status: detail.pr.status,
    note: detail.pr.note,
    changed_keys: detail.changed_keys ?? {},
    old_snap: baseRev?.snapshot ?? {},
    new_snap: detail.pr.snapshot ?? {},
    names: detail.names
  }
}

watch(
  () => props.status,
  () => {
    if (props.status === 'pending') {
      details.value = undefined
    }
  }
)
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between">
      <div class="flex flex-wrap items-center gap-2 text-sm">
        <KunAvatar :user="pr.user" />
        <span>{{ pr.user.name }} 提出更新请求</span>
        <span class="text-default-500">
          <KunTime :time="pr.created" />
        </span>
      </div>

      <div class="flex items-center gap-1 text-sm">
        <span
          class="flex items-center gap-1"
          :class="statusColorMap[pr.status]"
        >
          <span v-if="pr.completed_time">
            <KunTime :time="pr.completed_time" type="datetime" show-year />
          </span>
          <KunIcon :name="iconMap[pr.status]" />
          <span>
            {{ KUN_GALGAME_RESOURCE_PULL_REQUEST_STATUS_MAP[pr.status] }}
          </span>
        </span>

        <KunButton
          size="sm"
          variant="flat"
          v-if="!details"
          @click="handleGetDetails(pr.id)"
          :loading="isFetching"
        >
          {{ pr.status === 0 ? '查看 / 处理' : '查看差异' }}
        </KunButton>

        <KunButton
          :is-icon-only="true"
          variant="light"
          color="default"
          rounded="full"
          v-if="details"
          @click="details = undefined"
        >
          <KunIcon name="lucide:x" />
        </KunButton>
      </div>
    </div>

    <p v-if="pr.note" class="text-default-600 text-sm">
      <span class="text-default-400">说明: </span>{{ pr.note }}
    </p>

    <GalgamePrDetails v-if="details" :details="details" :refresh="refresh" />
  </div>
</template>
