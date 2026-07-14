<script setup lang="ts">
// Community-primitive toolset comment section (charter step 08). Mounted ONLY when
// the NUXT_PUBLIC_RESOURCE_COMMENTS_COMMUNITY flag is on — the single fork lives in
// ToolsetDetail.vue, so with the flag off this never renders and the legacy
// ToolsetCommentContainer is byte-identical.
//
// The read is a flat keyset "load more" growth list grouped into two tiers (root +
// one flat reply group), mirroring the galgame community container. The toolset
// owner's delete authority is a server-side superset (charter ruling 20), so the
// container needs no owner-id — the UI shows delete on author||canModerate only.
const props = defineProps<{
  toolsetId: number
}>()

const PAGE_LIMIT = 30

const posts = ref<GalgameCommunityComment[]>([])
const total = ref(0)
const nextCursor = ref('')
const seeded = ref(false)
const loadingMore = ref(false)

const { data, status } = await useKunFetch<GalgameCommunityCommentPage>(
  `/toolset/${props.toolsetId}/comments`,
  { lazy: true, method: 'GET', query: { limit: PAGE_LIMIT } }
)

const seedFrom = (page: GalgameCommunityCommentPage) => {
  posts.value = [...page.posts]
  total.value = page.total
  nextCursor.value = page.next_cursor
  seeded.value = true
}

// Check the current value first, then arm a non-immediate watch (the step-04 TDZ
// lesson — no self-stopping immediate watch).
if (data.value && !seeded.value) {
  seedFrom(data.value)
}
watch(data, (page) => {
  if (page && !seeded.value) {
    seedFrom(page)
  }
})

const hasMore = computed(() => nextCursor.value !== '')

const loadMore = async () => {
  if (!hasMore.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const page = await kunFetch<GalgameCommunityCommentPage>(
    `/toolset/${props.toolsetId}/comments`,
    { method: 'GET', query: { cursor: nextCursor.value, limit: PAGE_LIMIT } }
  )
  loadingMore.value = false
  if (page) {
    const seen = new Set(posts.value.map((p) => p.id))
    posts.value = [...posts.value, ...page.posts.filter((p) => !seen.has(p.id))]
    nextCursor.value = page.next_cursor
    total.value = page.total
  }
}

// Two-tier grouping: root + one flat reply group (a reply's root always precedes
// it in forward keyset order; a deep-cursor orphan renders as its own group).
interface CommentGroup {
  root: GalgameCommunityComment
  replies: GalgameCommunityComment[]
}

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

const isEmpty = computed(() => seeded.value && total.value === 0)

const handleNewComment = (post: GalgameCommunityComment) => {
  if (posts.value.some((p) => p.id === post.id)) {
    return
  }
  posts.value = [...posts.value, post]
  total.value += 1
}

const handleUpdated = (updated: GalgameCommunityComment) => {
  // The edit response is built without the target_user enrichment (the server's
  // UpdateComment returns buildCommunityItem directly). An edit changes only the
  // body, never the reply relationship, so keep the prior "A → B" target.
  posts.value = posts.value.map((p) =>
    p.id === updated.id
      ? { ...updated, target_user: updated.target_user ?? p.target_user }
      : p
  )
}

const handleTombstoned = (postId: number) => {
  posts.value = posts.value.map((p) =>
    p.id === postId
      ? { ...p, deleted: true, held: false, content: '', content_html: '' }
      : p
  )
}
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="评论"
      description="如果您对该工具有任何的使用疑问, 欢迎发布评论"
      scale="h2"
    />

    <ToolsetCommentCommunityComposer
      :toolset-id="toolsetId"
      @submitted="handleNewComment"
    />

    <KunLoading v-if="status === 'pending' && !seeded" />

    <KunNull v-else-if="isEmpty" />

    <div v-else-if="groups.length" class="space-y-6">
      <ToolsetCommentCommunityComment
        v-for="group in groups"
        :key="group.root.id"
        :comment="group.root"
        :replies="group.replies"
        :toolset-id="toolsetId"
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

    <!-- Single community-comment flag modal (reused, region agnostic). -->
    <GalgameCommentCommunityFlagModal />
  </div>
</template>
