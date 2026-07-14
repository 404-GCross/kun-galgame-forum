<script setup lang="ts">
// Community-primitive website comment section (charter step 08). Mounted ONLY when
// the NUXT_PUBLIC_RESOURCE_COMMENTS_COMMUNITY flag is on — the single fork lives in
// pages/website/[domain].vue, so with the flag off this never renders and the
// legacy WebsiteCommentContainer is byte-identical.
//
// The read is a flat keyset "load more" growth list grouped into two tiers (root +
// one flat reply group), mirroring the galgame community container. The addressing
// website_id rides the query; the :domain path segment is decorative (read from
// the route — this section only ever renders on /website/[domain]).
const props = defineProps<{
  websiteId: number
}>()

const route = useRoute()
const domain = computed(() => (route.params as { domain: string }).domain)

const PAGE_LIMIT = 30

const posts = ref<GalgameCommunityComment[]>([])
const total = ref(0)
const nextCursor = ref('')
const seeded = ref(false)
const loadingMore = ref(false)

const { data, status } = await useKunFetch<GalgameCommunityCommentPage>(
  `/website/${domain.value}/comments`,
  { lazy: true, method: 'GET', query: { website_id: props.websiteId, limit: PAGE_LIMIT } }
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
    `/website/${domain.value}/comments`,
    {
      method: 'GET',
      query: {
        website_id: props.websiteId,
        cursor: nextCursor.value,
        limit: PAGE_LIMIT
      }
    }
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

// Scroll to + highlight a just-published comment (parity with _scrollIntoComment).
const scrollToPost = (postId: number) => {
  nextTick(() => {
    setTimeout(() => {
      const el = document.getElementById(`website-comment-${postId}`)
      if (!el) {
        return
      }
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
      el.classList.add('bg-default-500/20')
      setTimeout(() => el.classList.remove('bg-default-500/20'), 2000)
    }, 200)
  })
}

const handleNewComment = (post: GalgameCommunityComment) => {
  if (posts.value.some((p) => p.id === post.id)) {
    return
  }
  posts.value = [...posts.value, post]
  total.value += 1
  // Only a new root gets scrolled to (parity — the old area scrolled roots).
  if (post.root_comment_id == null) {
    scrollToPost(post.id)
  }
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
  <KunCard :is-transparent="false" :is-hoverable="false" class-name="p-6">
    <h2 class="text-default-900 mb-6 text-2xl font-bold">用户评论</h2>

    <div class="mb-8">
      <WebsiteCommentCommunityComposer
        :website-id="websiteId"
        :domain="domain"
        @submitted="handleNewComment"
      />
    </div>

    <KunLoading v-if="status === 'pending' && !seeded" />

    <KunNull v-else-if="isEmpty" />

    <div v-else-if="groups.length" class="space-y-6">
      <WebsiteCommentCommunityComment
        v-for="group in groups"
        :key="group.root.id"
        :comment="group.root"
        :replies="group.replies"
        :website-id="websiteId"
        :domain="domain"
        :depth="0"
        @reply-added="handleNewComment"
        @tombstoned="handleTombstoned"
      />
    </div>

    <KunButton
      v-if="hasMore"
      variant="light"
      color="primary"
      full-width
      class-name="mt-6"
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多评论
    </KunButton>

    <!-- Single community-comment flag modal (reused, region agnostic). -->
    <GalgameCommentCommunityFlagModal />
  </KunCard>
</template>
