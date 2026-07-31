<script setup lang="ts">
// Publish wizard — the first stop in the "发布 Galgame" flow. Goal: keep
// duplicate submissions out of the moderation queue by surfacing existing
// records before the user fills out a full form.
//
// Two paths only (VNDB-id precise lookup was removed — the wiki has the
// full VNDB set synced as status=2 drafts, so a name search already
// surfaces them; users don't need to know VNDB ids):
//   1. Search by name → resolve to one of:
//        status=0 已发布   → 前往发布资源 (/galgame/:gid)
//        status=2 草稿      → 认领并发布 (POST /:gid/claim, +3 萌萌点)
//        status=3/4 自己的草稿 → 继续编辑 (/edit/galgame/draft/:gid)
//   2. Nothing matches → 新建申请 → /edit/galgame/create
//
// `items` now comes off the catalog registry (claim_state=live,draft) while
// `pending` — the caller's OWN submissions — still comes off the wiki face.
// One consequence is visible here: the registry cannot tell an unclaimed VNDB
// draft apart from someone else's submission awaiting review, so both arrive as
// status=2 and the 认领 attempt is what discovers the difference (the wiki
// refuses it). The copy below says so rather than pretending otherwise.

interface SearchHit {
  id: number
  vndb_id?: string
  name_zh_cn?: string
  name_ja_jp?: string
  name_en_us?: string
  name_zh_tw?: string
  banner?: string
  // K-PR6: banner_image_hash retired in wiki PR5; effective_banner_hash
  // is the derived banner source (= covers[sort_order=0].image_hash).
  effective_banner_hash?: string
  status?: number
}

interface WizardSearchResp {
  items: SearchHit[]
  pending?: SearchHit[]
  total: number
}

const q = ref('')
const hasSearched = ref(false)
const isSearching = ref(false)
const searchResults = ref<WizardSearchResp | null>(null)

// status badge + wire-name resolution are shared (shared/utils/
// galgameStatus.ts). Fallback (VNDB id / #id) computed per call site.
const nameOfHit = (h: SearchHit): string =>
  galgameNameFromWire(h, h.vndb_id ? `VNDB ${h.vndb_id}` : `#${h.id}`)

const statusBadge = galgameStatusBadge

const handleSearch = async () => {
  if (!q.value.trim()) {
    useMessage('请先输入关键词', 'warn')
    return
  }
  isSearching.value = true
  // /galgame/search/wizard forces include_pending=true server-side; the
  // Bearer is attached automatically by the session middleware so wiki
  // resolves the caller's own pending list.
  const res = await kunFetch<WizardSearchResp>('/galgame/search/wizard', {
    method: 'GET',
    query: { q: q.value.trim(), limit: 12 }
  })
  isSearching.value = false
  hasSearched.value = true
  searchResults.value = res
}

const isClaiming = ref(false)

const handleClaim = async (gid: number) => {
  const ok = await useComponentMessageStore().alert(
    '认领此草稿吗?',
    '认领后该条目立即变为已发布状态, 您将成为该 Galgame 的创建者, 并获得 +3 萌萌点。若该条目其实是他人正在审核中的投稿, 认领会被拒绝。'
  )
  if (!ok) return

  isClaiming.value = true
  const result = await kunFetch<{ id: number }>(`/galgame/${gid}/claim`, {
    method: 'POST',
    body: {}
  })
  isClaiming.value = false
  if (result?.id) {
    useKunLoliInfo('认领成功, 已发布', 5)
    await navigateTo(`/galgame/${result.id}`)
  }
}

// Carry the typed name over to the create form so the user doesn't
// re-type it.
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

// Arrived from a calendar "未发布" card (/edit/galgame/publish?q=<name>):
// pre-fill + auto-search so the clicked draft surfaces straight away for 认领.
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
        <KunLink to="/edit/galgame/mine">
          <KunButton size="sm" variant="flat">我的提交</KunButton>
        </KunLink>
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
        搜索覆盖已发布的 Galgame 及尚未发布的草稿 (可一键认领),
        同时会显示您自己的待审核 / 已拒绝草稿。草稿中可能包含他人正在审核中的投稿,
        这类条目无法被认领。
      </p>
    </div>

    <div v-if="searchResults" class="space-y-4">
      <!-- pending: own status=3/4 hits, most actionable, shown first -->
      <div
        v-if="searchResults.pending && searchResults.pending.length"
        class="space-y-2"
      >
        <h3 class="text-default-700 text-sm font-bold">您的待审 / 已拒草稿</h3>
        <div
          v-for="hit in searchResults.pending"
          :key="`pending-${hit.id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="truncate font-medium">{{ nameOfHit(hit) }}</h4>
              <KunChip
                size="xs"
                variant="flat"
                :color="statusBadge(hit.status).color"
              >
                {{ statusBadge(hit.status).label }}
              </KunChip>
            </div>
            <p class="text-default-500 text-sm">
              VNDB: {{ hit.vndb_id || '—' }}
            </p>
          </div>
          <KunLink :to="`/edit/galgame/draft/${hit.id}`">
            <KunButton size="sm" variant="flat">继续编辑</KunButton>
          </KunLink>
        </div>
      </div>

      <!--
        items: search hits. The registry also surfaces unpublished
        entries (status=2, projected from claim_state=draft), so the
        action MUST branch on status:
          status=0 已发布 → 前往发布资源
          status=2 VNDB 草稿 → 认领并发布 (status=2 is NOT viewable via
            /galgame/:gid, a blanket detail link would 404)
          status=3/4 → 继续编辑
      -->
      <div v-if="searchResults.items.length" class="space-y-2">
        <h3 class="text-default-700 text-sm font-bold">匹配的 Galgame</h3>
        <div
          v-for="hit in searchResults.items"
          :key="`item-${hit.id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <KunImage
            v-if="hit.banner"
            :src="hit.banner"
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
                :color="statusBadge(hit.status).color"
              >
                {{ statusBadge(hit.status).label }}
              </KunChip>
            </div>
            <p class="text-default-500 text-sm">
              VNDB: {{ hit.vndb_id || '—' }}
            </p>
          </div>
          <KunButton
            v-if="hit.status === GalgameStatus.VndbDraft"
            size="sm"
            :loading="isClaiming"
            :disabled="isClaiming"
            @click="handleClaim(hit.id)"
          >
            认领并发布
          </KunButton>
          <KunLink
            v-else-if="
              hit.status === GalgameStatus.Pending ||
              hit.status === GalgameStatus.Declined
            "
            :to="`/edit/galgame/draft/${hit.id}`"
          >
            <KunButton size="sm" variant="flat">继续编辑</KunButton>
          </KunLink>
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
      <KunButton size="lg" @click="handleCreateNew">新建 Galgame 申请</KunButton>
    </div>
  </div>
</template>
