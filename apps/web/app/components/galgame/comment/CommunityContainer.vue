<script setup lang="ts">
// Community-primitive comment section (charter step 04). Mounted ONLY when the
// NUXT_PUBLIC_GALGAME_COMMENTS_COMMUNITY flag is on — the single fork lives in
// Galgame.vue, so with the flag off this component never renders and the legacy
// GalgameCommentContainer is byte-identical.
//
// The list is a keyset "load more" growth list: page 1 comes from the resolve
// (get-or-create), later pages from the cursor. Posts accumulate flat and are
// grouped into two tiers (root + one flat reply group) for rendering.
const emit = defineEmits<{
  // Surface the (lazy, client-side) fetch state so the tab panel dims while it
  // loads — same contract as the legacy container.
  'update:loading': [boolean]
}>()

const route = useRoute()
const config = useRuntimeConfig()
const gid = parseInt((route.params as { gid: string }).gid)

// See GalgameResource: for a wiki-catalogue game the forum hasn't ingested, hide
// the empty-state (the detail page's 未收录 notice covers it) but KEEP the
// composer — commenting creates the local row, part of the recording funnel.
const galgame = inject<GalgameDetail>('galgame')

const PAGE_LIMIT = 30

const posts = ref<GalgameCommunityComment[]>([])
const total = ref(0)
const nextCursor = ref('')
const seeded = ref(false)
const loadingMore = ref(false)

// Page 1 via useKunFetch so SSR + hydration + the tab-dim `status` all work.
const { data, status } = await useKunFetch<GalgameCommunityCommentPage>(
  `/galgame/${gid}/comments`,
  { lazy: true, method: 'GET', query: { limit: PAGE_LIMIT } }
)
watchEffect(() => emit('update:loading', status.value === 'pending'))

// Seed local state once page 1 lands; guarded so later optimistic mutations are
// never clobbered by a re-run.
watch(
  data,
  (page) => {
    if (!page || seeded.value) {
      return
    }
    posts.value = [...page.posts]
    total.value = page.total
    nextCursor.value = page.next_cursor
    seeded.value = true
  },
  { immediate: true }
)

const hasMore = computed(() => nextCursor.value !== '')

const loadMore = async () => {
  if (!hasMore.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const page = await kunFetch<GalgameCommunityCommentPage>(
    `/galgame/${gid}/comments`,
    { method: 'GET', query: { cursor: nextCursor.value, limit: PAGE_LIMIT } }
  )
  loadingMore.value = false
  if (page) {
    // De-dup against optimistic inserts already present in the list.
    const seen = new Set(posts.value.map((p) => p.id))
    posts.value = [...posts.value, ...page.posts.filter((p) => !seen.has(p.id))]
    nextCursor.value = page.next_cursor
    total.value = page.total
  }
}

// ──────────────────────────────────────────
// Two-tier grouping
// ──────────────────────────────────────────

interface CommentGroup {
  root: GalgameCommunityComment
  replies: GalgameCommunityComment[]
}

// Group the flat, arrival-ordered list into root + one flat reply group. Because
// post_number is monotonic and a root always precedes its replies in ascending
// keyset order, a reply's root is already present in forward pagination — so
// orphans don't occur in normal flow. Defensively, an orphan reply (deep-cursor
// edge) renders as its own standalone group at its arrival position.
const groups = computed<CommentGroup[]>(() => {
  const list: CommentGroup[] = []
  const byRootId = new Map<number, CommentGroup>()
  for (const p of posts.value) {
    if (p.root_comment_id == null) {
      const group: CommentGroup = { root: p, replies: [] }
      byRootId.set(p.id, group)
      list.push(group)
    } else {
      const owner = byRootId.get(p.root_comment_id)
      if (owner) {
        owner.replies.push(p)
      } else {
        list.push({ root: p, replies: [] })
      }
    }
  }
  return list
})

const isEmpty = computed(
  () => seeded.value && total.value === 0 && galgame?.is_on_forum !== false
)

// ──────────────────────────────────────────
// Optimistic updates
// ──────────────────────────────────────────

const handleNewComment = (post: GalgameCommunityComment) => {
  if (posts.value.some((p) => p.id === post.id)) {
    return
  }
  // Append in arrival order: a new root lands at the end; a new reply groups
  // under its (already-present) root. total tracks the thread's posts_count.
  posts.value = [...posts.value, post]
  total.value += 1
}

const handleUpdated = (updated: GalgameCommunityComment) => {
  posts.value = posts.value.map((p) => (p.id === updated.id ? updated : p))
}

const handleTombstoned = (postId: number) => {
  // Tombstone keeps the floor: flip the row to deleted, blank the body. The
  // thread's posts_count is unchanged (charter ruling 11), so total stays.
  posts.value = posts.value.map((p) =>
    p.id === postId
      ? { ...p, deleted: true, held: false, content: '', content_html: '' }
      : p
  )
}

// ──────────────────────────────────────────
// Deep-link continuity (charter ruling 9)
// ──────────────────────────────────────────

const scrollToPost = (postId: number) => {
  nextTick(() => {
    setTimeout(() => {
      const el = document.getElementById(`galgame-comment-${postId}`)
      if (!el) {
        return
      }
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
      const ring = ['outline-2', 'outline-offset-2', 'outline-primary', 'rounded-lg']
      el.classList.add(...ring)
      setTimeout(() => el.classList.remove(...ring), 3000)
    }, 200)
  })
}

// Silent legacy-id resolve: a 404 (old content not imported yet) is ignored,
// so a raw $fetch swallows it instead of kunFetch's error toast.
const locateLegacy = async (legacyId: number): Promise<number | null> => {
  try {
    const resp = await $fetch<{ code: number; data?: { post_id: number } }>(
      `${config.public.apiBaseUrl}/api/galgame/${gid}/comments/locate`,
      { query: { legacy_id: legacyId }, credentials: 'include' }
    )
    return resp && resp.code === 0 && resp.data ? resp.data.post_id : null
  } catch {
    return null
  }
}

// Page forward until the target post is loaded (or the list is exhausted), then
// scroll. Capped so a bad id can't run away.
const ensureLoadedAndScroll = async (postId: number) => {
  let guard = 0
  while (
    !posts.value.some((p) => p.id === postId) &&
    hasMore.value &&
    guard < 50
  ) {
    await loadMore()
    guard += 1
  }
  if (posts.value.some((p) => p.id === postId)) {
    scrollToPost(postId)
  }
}

const resolveDeepLink = async () => {
  const commentParam = Number(route.query.comment) || 0
  // The old @-mention notification shape carried a `thread` param alongside a
  // legacy comment id; its presence flags the id as legacy.
  const hasThreadParam = route.query.thread != null
  const hashMatch = (route.hash || '').match(/^#galgame-comment-(\d+)$/)

  if (commentParam) {
    if (hasThreadParam) {
      const postId = await locateLegacy(commentParam)
      if (postId) {
        await ensureLoadedAndScroll(postId)
      }
      return
    }
    // New deep-link: `comment` IS the community post id.
    await ensureLoadedAndScroll(commentParam)
    return
  }

  if (hashMatch) {
    const raw = Number(hashMatch[1])
    // Old external anchors (#galgame-comment-<id>) may carry either a new post
    // id or a legacy id: try it as a post id first, then fall back to locate.
    let guard = 0
    while (!posts.value.some((p) => p.id === raw) && hasMore.value && guard < 50) {
      await loadMore()
      guard += 1
    }
    if (posts.value.some((p) => p.id === raw)) {
      scrollToPost(raw)
      return
    }
    const mapped = await locateLegacy(raw)
    if (mapped) {
      await ensureLoadedAndScroll(mapped)
    }
  }
}

onMounted(() => {
  // Resolve deep-links client-side, once the first page has seeded.
  const stop = watch(
    seeded,
    (ready) => {
      if (ready) {
        stop()
        resolveDeepLink()
      }
    },
    { immediate: true }
  )
})
</script>

<template>
  <div class="space-y-3">
    <KunHeader name="游戏评论" scale="h2">
      <template #endContent>
        <KunLink size="sm" to="/topic/1482">
          Galgame 评论注意事项, 资源失效, 解压密码错误等问题反馈
        </KunLink>
      </template>
    </KunHeader>

    <GalgameCommentCommunityComposer
      :galgame-id="gid"
      @submitted="handleNewComment"
    />

    <KunLoading v-if="status === 'pending'" />

    <KunNull
      v-else-if="isEmpty"
      description="没人评论, 是没人要这个 Galgame 的小只可爱软萌女孩子了吗, 呜呜呜呜呜呜！！"
    />

    <div v-else-if="groups.length" class="space-y-6">
      <GalgameCommentCommunityComment
        v-for="group in groups"
        :key="group.root.id"
        :comment="group.root"
        :replies="group.replies"
        :galgame-id="gid"
        :depth="0"
        @reply-added="handleNewComment"
        @updated="handleUpdated"
        @tombstoned="handleTombstoned"
      />
    </div>

    <KunButton
      v-if="hasMore"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多评论
    </KunButton>

    <!-- Single community-comment flag modal for this section. -->
    <GalgameCommentCommunityFlagModal />
  </div>
</template>
