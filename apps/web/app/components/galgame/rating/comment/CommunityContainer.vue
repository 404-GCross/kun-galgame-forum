<script setup lang="ts">
// Community-primitive rating comment section (charter step 08). Mounted ONLY when
// the NUXT_PUBLIC_RESOURCE_COMMENTS_COMMUNITY flag is on — the single fork lives in
// GalgameRatingDetailDetail.vue, so with the flag off this never renders and the
// legacy GalgameRatingCommentContainer is byte-identical.
//
// The rating area is FLAT (no grouping): the read is a keyset "load more" growth
// list and every post is a root with an explicit target_user. This mirrors the
// galgame community container's paging machinery, minus the two-tier grouping.
const props = defineProps<{
  ratingId: number
  ratingAuthor: KunUser
}>()

const PAGE_LIMIT = 30

const posts = ref<GalgameCommunityComment[]>([])
const total = ref(0)
const nextCursor = ref('')
const seeded = ref(false)
const loadingMore = ref(false)

const { data, status } = await useKunFetch<GalgameCommunityCommentPage>(
  `/galgame-rating/${props.ratingId}/comments`,
  { lazy: true, method: 'GET', query: { limit: PAGE_LIMIT } }
)

const seedFrom = (page: GalgameCommunityCommentPage) => {
  posts.value = [...page.posts]
  total.value = page.total
  nextCursor.value = page.next_cursor
  seeded.value = true
}

// Seed once page 1 lands. Check the current value first (SSR/hydrated payloads are
// already present in setup), then arm a NON-immediate watch for the lazy case — no
// self-stopping immediate watch (the step-04 TDZ lesson).
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
    `/galgame-rating/${props.ratingId}/comments`,
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
  <KunCard :is-transparent="false" :is-hoverable="false">
    <KunHeader
      name="评论区"
      description="发布对这个评分的观点, 请不要锐评"
      scale="h2"
    />

    <div class="space-y-3">
      <GalgameRatingCommentCommunityComposer
        :rating-id="ratingId"
        :target-user-id="ratingAuthor.id"
        @submitted="handleNewComment"
      />

      <KunLoading v-if="status === 'pending' && !seeded" />

      <KunNull v-else-if="isEmpty" />

      <div v-else-if="posts.length" class="space-y-3">
        <GalgameRatingCommentCommunityComment
          v-for="post in posts"
          :key="post.id"
          :comment="post"
          :rating-id="ratingId"
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
    </div>

    <!-- Single community-comment flag modal for this section (reused, region
         agnostic — post-id addressed). -->
    <GalgameCommentCommunityFlagModal />
  </KunCard>
</template>
