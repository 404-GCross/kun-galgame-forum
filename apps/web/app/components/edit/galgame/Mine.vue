<script setup lang="ts">
const pageData = reactive({
  before: 0,
  limit: 20
})

const { data, status, refresh } = await useKunFetch<UserClaimList>(
  '/galgame/mine',
  { query: pageData }
)

const stateBadge = galgameClaimStateBadge
const nameOf = (item: UserClaimItem) => item.display_name || '(无标题)'
const gidOf = (item: UserClaimItem) => item.product_work_id ?? 0

const isWithdrawing = ref<Record<number, boolean>>({})

const handleWithdraw = async (item: UserClaimItem) => {
  const ok = await useComponentMessageStore().alert(
    '确定撤回这条申请吗?',
    '撤回后该条目会退回草稿状态, 不再公开展示。您填写的内容不会丢失, 随时可以重新提交。'
  )
  if (!ok) {
    return
  }
  const gid = gidOf(item)
  isWithdrawing.value = { ...isWithdrawing.value, [gid]: true }
  const res = await kunFetch<string>(`/galgame/${gid}`, {
    method: 'DELETE'
  })
  isWithdrawing.value = { ...isWithdrawing.value, [gid]: false }
  if (res !== null) {
    useMessage('已撤回', 'success')
    refresh()
  }
}

const isResubmitting = ref<Record<number, boolean>>({})

const handleResubmit = async (item: UserClaimItem) => {
  const gid = gidOf(item)
  isResubmitting.value = { ...isResubmitting.value, [gid]: true }
  const res = await kunFetch<unknown>(`/galgame/${gid}/resubmit`, {
    method: 'POST'
  })
  isResubmitting.value = { ...isResubmitting.value, [gid]: false }
  if (res !== null) {
    useMessage('已重新提交审核', 'success')
    refresh()
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- 铁律 #11: keep this template's SINGLE real root element. A sibling
         or a leading comment at the template root is itself a root node,
         trips Nuxt's "does not have a single root node" warning, and
         silently drops the page-transition enter animation. -->
    <KunHeader
      name="我的 Galgame 提交"
      description="您提交的 Galgame 申请, 审核中 / 已拒绝 / 草稿都会显示在此处。审核通过的 Galgame 会成为公开条目, 不再列在这里。"
    >
      <template #endContent>
        <div class="flex gap-2">
          <KunLink to="/edit/galgame/publish">
            <KunButton size="sm">新建提交</KunButton>
          </KunLink>
          <KunLink to="/message/notice">
            <KunButton size="sm" variant="flat">审核通知</KunButton>
          </KunLink>
        </div>
      </template>
    </KunHeader>

    <KunDivider />

    <KunInfo
      v-if="!data"
      color="danger"
      title="加载失败"
      description="无法获取您的提交列表, 可能是后端 / Galgame 资料库暂时不可用, 请稍后重试。"
    />

    <div v-else-if="data.items.length" class="flex flex-col gap-3">
      <div
        v-for="item in data.items"
        :key="item.work_id"
        class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
      >
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <h3
              class="hover:text-primary truncate text-lg font-medium transition-colors"
            >
              {{ nameOf(item) }}
            </h3>
            <KunChip
              size="xs"
              variant="flat"
              :color="stateBadge(item.claim_state).color"
            >
              {{ stateBadge(item.claim_state).label }}
            </KunChip>
          </div>
          <div
            class="text-default-500 flex flex-wrap items-center gap-2 text-sm"
          >
            <span>提交于 <KunTime :time="item.first_acted_at" /></span>
            <template v-if="item.last_event_at !== item.first_acted_at">
              <span>·</span>
              <span>最后处理 <KunTime :time="item.last_event_at" /></span>
            </template>
          </div>
          <div
            v-if="item.claim_state === CLAIM_STATE_DECLINED && item.last_reason"
            class="text-danger bg-danger/10 mt-1 rounded-md px-2 py-1 text-sm"
          >
            被拒原因: {{ item.last_reason }}
          </div>
        </div>
        <div class="flex shrink-0 gap-2">
          <KunLink :to="`/galgame/${gidOf(item)}/edit`">
            <KunButton size="sm" variant="flat">编辑</KunButton>
          </KunLink>
          <KunButton
            v-if="item.claim_state === CLAIM_STATE_DECLINED"
            size="sm"
            color="primary"
            variant="flat"
            :loading="isResubmitting[gidOf(item)]"
            :disabled="isResubmitting[gidOf(item)]"
            @click="handleResubmit(item)"
          >
            重新提交
          </KunButton>
          <KunButton
            v-else-if="item.claim_state !== CLAIM_STATE_DRAFT"
            size="sm"
            color="danger"
            variant="flat"
            :loading="isWithdrawing[gidOf(item)]"
            :disabled="isWithdrawing[gidOf(item)]"
            @click="handleWithdraw(item)"
          >
            撤回
          </KunButton>
        </div>
      </div>
    </div>

    <KunNull v-else-if="data && !data.items.length" />

    <KunButton
      v-if="data && data.next_before"
      variant="flat"
      :loading="status === 'pending'"
      @click="pageData.before = data.next_before"
    >
      加载更多
    </KunButton>
  </div>
</template>
