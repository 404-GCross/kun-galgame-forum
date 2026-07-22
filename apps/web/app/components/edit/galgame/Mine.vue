<script setup lang="ts">
// "我的提交" — lists the current user's pending/declined galgame drafts
// served by GET /api/galgame/mine. This is the post-submit landing page
// (replacing the legacy redirect to /galgame/:gid, which would 404 for
// the submitter while status=3 is anonymously invisible).
//
// See docs/galgame_wiki/07-submission.md §GET /galgame/mine for the
// upstream wire format. Kungal forwards verbatim — no enrichment.
//
// TEMPLATE SINGLE-ROOT: the root <KunCard> renders unconditionally and
// must be the *only* top-level template node — NOT preceded by a
// template comment. In dev, Vue keeps template comments as comment
// vnodes, so a leading `<!-- ... -->` becomes a sibling of KunCard,
// making the page's render root a 2-child Fragment and tripping Nuxt's
// "does not have a single root node" route-transition warning. A bare
// `v-if="data"` root has the same effect (empty comment vnode on
// fetch failure). Hence: all conditional content lives INSIDE the card.

const pageData = reactive({
  // Default 3,4 = pending + declined. Approved drafts disappear from
  // this view since they're publicly listable as normal galgames.
  status: '3,4',
  page: 1,
  limit: 20
})

const { data, status, refresh } = await useKunFetch<MineGalgameList>(
  '/galgame/mine',
  { query: pageData }
)

// status badge + wire-name resolution are shared (shared/utils/
// galgameStatus.ts) so this list, the wizard, the draft editor and the
// admin queue can't drift apart.
const statusBadge = galgameStatusBadge
const nameOf = (item: MineGalgameItem) => galgameNameFromWire(item, '(无标题)')

const isWithdrawing = ref<Record<number, boolean>>({})

const handleWithdraw = async (item: MineGalgameItem) => {
  const ok = await useComponentMessageStore().alert(
    '确定撤回这条申请吗?',
    '撤回后将不可恢复, 您仍然可以重新提交一份新的申请。'
  )
  if (!ok) {
    return
  }
  isWithdrawing.value = { ...isWithdrawing.value, [item.id]: true }
  const res = await kunFetch<string>(`/galgame/${item.id}`, {
    method: 'DELETE'
  })
  isWithdrawing.value = { ...isWithdrawing.value, [item.id]: false }
  if (res !== null) {
    useMessage('已撤回', 'success')
    refresh()
  }
}
</script>

<template>
  <div class="space-y-4">
    <KunHeader
      name="我的 Galgame 提交"
      description="您提交的 Galgame 申请, 待审核 / 已拒绝的草稿都会显示在此处。审核通过的 Galgame 会成为公开条目, 不再列在这里。"
    >
      <template #endContent>
        <div class="flex gap-2">
          <KunLink to="/edit/galgame/publish">
            <KunButton size="sm">新建提交</KunButton>
          </KunLink>
          <KunLink to="/message/wiki">
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
        :key="item.id"
        class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
      >
        <KunImage
          :src="getEffectiveBanner(item, { variant: 'mini' }) || '/kungalgame.webp'"
          loading="lazy"
          placeholder="/placeholder.webp"
          class="h-20 w-32 shrink-0 rounded object-cover"
          :style="{ aspectRatio: '16/9' }"
        />
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
              :color="statusBadge(item.status).color"
            >
              {{ statusBadge(item.status).label }}
            </KunChip>
          </div>
          <div class="text-default-500 flex flex-wrap items-center gap-2 text-sm">
            <span>VNDB: {{ item.vndb_id || '—' }}</span>
            <span>·</span>
            <span>提交于 <KunTime :time="item.created" /></span>
            <template v-if="item.updated !== item.created">
              <span>·</span>
              <span>最后修改 <KunTime :time="item.updated" /></span>
            </template>
          </div>
          <div
            v-if="
              item.status === GalgameStatus.Declined && item.decline_reason
            "
            class="text-danger bg-danger/10 mt-1 rounded-md px-2 py-1 text-sm"
          >
            被拒原因: {{ item.decline_reason }}
          </div>
        </div>
        <div class="flex shrink-0 gap-2">
          <KunLink :to="`/edit/galgame/draft/${item.id}`">
            <KunButton size="sm" variant="flat">编辑</KunButton>
          </KunLink>
          <KunButton
            size="sm"
            color="danger"
            variant="flat"
            :loading="isWithdrawing[item.id]"
            :disabled="isWithdrawing[item.id]"
            @click="handleWithdraw(item)"
          >
            撤回
          </KunButton>
        </div>
      </div>
    </div>

    <KunNull v-else-if="data && !data.items.length" />

    <KunPagination
      v-if="data && data.total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
