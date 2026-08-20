<script setup lang="ts">
interface SearchHit {
  id: number
  work_id?: number
  vndb_id?: string
  name?: string
  effective_banner_hash?: string
  effective_banner_url?: string
  claim_state?: string
}

interface WizardSearchResp {
  items: SearchHit[]
  pending?: UserClaimItem[]
  total: number
}

const q = ref('')
const canReviewClaims = useCan('galgame.claim.review')
const hasSearched = ref(false)
const isSearching = ref(false)
const searchResults = ref<WizardSearchResp | null>(null)

const nameOfHit = (h: SearchHit): string =>
  h.name || (h.vndb_id ? `VNDB ${h.vndb_id}` : `#${h.id}`)

const stateBadge = galgameClaimStateBadge

const handleSearch = async () => {
  if (!q.value.trim()) {
    useMessage('请先输入关键词', 'warn')
    return
  }
  isSearching.value = true
  const res = await kunFetch<WizardSearchResp>('/galgame/search/wizard', {
    method: 'GET',
    query: { q: q.value.trim(), limit: 12 }
  })
  isSearching.value = false
  hasSearched.value = true
  searchResults.value = res
}

const isClaiming = ref(false)

const handleClaim = async (hit: SearchHit) => {
  const unclaimed = isUnclaimedState(hit.claim_state)
  const ok = await useComponentMessageStore().alert(
    unclaimed ? '将此作品收录到本站吗?' : '认领此草稿吗?',
    unclaimed
      ? '该作品目前只存在于资料库, 本站尚未收录。收录后它会立即以已发布状态出现在本站, 您将成为该 Galgame 的创建者, 并获得 +3 萌萌点。'
      : '认领后该条目立即变为已发布状态, 您将成为该 Galgame 的创建者, 并获得 +3 萌萌点。若该条目其实是他人正在审核中的投稿, 认领会被拒绝。'
  )
  if (!ok) return

  const endpoint = unclaimed
    ? `/galgame/work/${hit.work_id}/claim`
    : `/galgame/${hit.id}/claim`

  isClaiming.value = true
  // gid comes back from the server: for an unclaimed row the forum id does not
  // exist until this call mints it, and work_id is catalog's id space, not the
  // forum's — the same conflation that once linked 我的审核 rows to an
  // unrelated galgame.
  const result = await kunFetch<{ to_state: string; gid?: number }>(endpoint, {
    method: 'POST',
    body: {}
  })
  isClaiming.value = false
  if (result?.to_state) {
    useKunLoliInfo(unclaimed ? '收录成功, 已发布' : '认领成功, 已发布', 5)
    await navigateTo(`/galgame/${result.gid ?? hit.id}`)
  }
}

const handleCreateNew = async () => {
  const store = usePersistEditGalgameStore()
  if (q.value.trim() && !store.name['zh-cn']) {
    store.name['zh-cn'] = q.value.trim()
  }
  await navigateTo('/edit/galgame/create')
}

const noMatches = computed(
  () =>
    hasSearched.value &&
    searchResults.value !== null &&
    !searchResults.value.items.length &&
    !searchResults.value.pending?.length
)

const route = useRoute()
onMounted(() => {
  const pre = route.query.q
  if (typeof pre === 'string' && pre.trim()) {
    q.value = pre.trim()
    handleSearch()
  }
})
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="发布 Galgame"
      description="先搜索您想发布的游戏：已存在的直接前往或一键认领，确实没有的再新建申请，避免重复提交。"
    >
      <template #endContent>
        <div class="flex items-center gap-2">
          <KunLink to="/edit/galgame/mine">
            <KunButton size="sm" variant="flat">我的提交</KunButton>
          </KunLink>
          <KunLink v-if="canReviewClaims" to="/edit/galgame/audited">
            <KunButton size="sm" variant="flat">我的审核</KunButton>
          </KunLink>
        </div>
      </template>
    </KunHeader>

    <KunDivider>
      <span class="mx-2">① 搜索是否已存在</span>
    </KunDivider>

    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <KunInput
          v-model="q"
          placeholder="输入游戏名 (任意语言)"
          @keydown.enter="handleSearch"
        />
        <KunButton
          class-name="whitespace-nowrap"
          :loading="isSearching"
          @click="handleSearch"
        >
          搜索
        </KunButton>
      </div>
      <p class="text-default-500 text-sm">
        搜索覆盖已发布的 Galgame、尚未发布的草稿 (可一键认领)、资料库中本站尚未收录的作品
        (标记为「未认领」, 可一键收录), 以及他人正在审核中的投稿 (标记为「审核中」,
        无法认领); 同时会显示您自己的待审核 / 已拒绝投稿。
      </p>
    </div>

    <div v-if="searchResults" class="space-y-4">
      <div
        v-if="searchResults.pending && searchResults.pending.length"
        class="space-y-2"
      >
        <h3 class="text-default-700 text-sm font-bold">您的待审 / 已拒草稿</h3>
        <div
          v-for="item in searchResults.pending"
          :key="`pending-${item.work_id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="truncate font-medium">
                {{ item.display_name || `#${item.work_id}` }}
              </h4>
              <KunChip
                size="xs"
                variant="flat"
                :color="stateBadge(item.claim_state).color"
              >
                {{ stateBadge(item.claim_state).label }}
              </KunChip>
            </div>
            <p v-if="item.last_reason" class="text-default-500 text-sm">
              {{ item.last_reason }}
            </p>
          </div>
          <KunLink
            v-if="galgameClaimGid(item)"
            :to="`/galgame/${galgameClaimGid(item)}/edit`"
          >
            <KunButton size="sm" variant="flat">继续编辑</KunButton>
          </KunLink>
        </div>
      </div>

      <div v-if="searchResults.items.length" class="space-y-2">
        <h3 class="text-default-700 text-sm font-bold">匹配的 Galgame</h3>
        <div
          v-for="hit in searchResults.items"
          :key="`item-${hit.work_id ?? hit.id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <KunImage
            v-if="hit.effective_banner_url"
            :src="hit.effective_banner_url"
            loading="lazy"
            placeholder="/placeholder.webp"
            class="h-16 w-28 shrink-0 rounded object-cover"
            :style="{ aspectRatio: '16/9' }"
          />
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="truncate font-medium">{{ nameOfHit(hit) }}</h4>
              <KunChip
                size="xs"
                variant="flat"
                :color="stateBadge(hit.claim_state).color"
              >
                {{ stateBadge(hit.claim_state).label }}
              </KunChip>
            </div>
            <p class="text-default-500 text-sm">
              VNDB: {{ hit.vndb_id || '—' }}
            </p>
          </div>
          <KunButton
            v-if="isClaimableState(hit.claim_state)"
            size="sm"
            :loading="isClaiming"
            :disabled="isClaiming"
            @click="handleClaim(hit)"
          >
            {{ isUnclaimedState(hit.claim_state) ? '收录并发布' : '认领并发布' }}
          </KunButton>
          <span
            v-else-if="hit.claim_state === CLAIM_STATE_PENDING"
            class="text-default-400 shrink-0 text-sm"
          >
            他人投稿审核中
          </span>
          <KunLink v-else :to="`/galgame/${hit.id}`">
            <KunButton size="sm" variant="flat">前往发布资源</KunButton>
          </KunLink>
        </div>
      </div>

      <KunInfo
        v-if="noMatches"
        color="info"
        title="没有找到匹配的 Galgame"
        description="确认确实没有后, 用下方「新建 Galgame 申请」提交。"
      />
    </div>

    <KunDivider>
      <span class="mx-2">② 都没有？新建申请</span>
    </KunDivider>

    <KunInfo
      color="info"
      title="提交一份新的 Galgame 申请"
      description="仅用于 VNDB 未收录的原创 / 同人 / 独立作品。提交后进入审核队列, 审核通过才会公开。"
    />
    <div class="flex justify-end">
      <KunButton size="lg" @click="handleCreateNew"
        >新建 Galgame 申请</KunButton
      >
    </div>
  </div>
</template>
