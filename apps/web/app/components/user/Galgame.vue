<script setup lang="ts">
import {
  kunUserGalgameNavItem,
  type KUN_USER_PAGE_GALGAME_TYPE
} from '~/constants/user'

// Single component, two rendering modes:
//
//   - galgame_like / galgame_favorite
//     → GET /user/:id/galgames returns galgame cards (cover + name +
//       counts). Same shape the rest of the site uses.
//
//   - galgame_comment / galgame_comment_target / galgame_comment_like
//     → GET /user/:id/galgame-comments returns comment rows
//       (content + author + created + parent galgame id). Rendered
//       in a minimal comment-card style — same UX as
//       /user/:id/comment/ for topic comments. Earlier this branch
//       reused the galgame-card endpoint so you saw the parent
//       galgame instead of the actual comment text.
const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_GALGAME_TYPE)[number]
}>()

const COMMENT_TYPES = [
  'galgame_comment',
  'galgame_comment_target',
  'galgame_comment_like'
] as const

const isCommentMode = computed(() =>
  (COMMENT_TYPES as readonly string[]).includes(props.type)
)

const activeTab = ref(props.type)
const pageData = reactive({
  page: 1,
  limit: 24,
  type: props.type,
  userId: props.userId
})

interface UserGalgameCommentItem {
  id: number
  galgameId: number
  content: string
  contentHtml: string
  user: { id: number; name: string; avatar: string }
  created: string
}

// Two parallel fetches gated by `isCommentMode`. We swap the endpoint
// (and the result shape) instead of branching client-side on a single
// union response — keeps the wire payload tight and the typing clean.
// Global "显示没有下载资源的 Galgame" preference (cookie-persisted, SSR-safe).
// Off (default) hides resource-less galgames; added to the query + watch so a
// toggle re-fetches. (Comment-mode fetch below is unaffected — it lists
// comments, not galgame cards.)
const settings = usePersistSettingsStore()
const { data: galgameData, status: galgameStatus } = await useKunFetch<{
  items: GalgameCard[]
  total: number
}>(() => `/user/${props.userId}/galgames`, {
  query: computed(() => ({
    ...pageData,
    showNoResource: settings.showKUNGalgameNoResource
  })),
  watch: [
    () => pageData.page,
    () => pageData.type,
    () => settings.showKUNGalgameNoResource
  ],
  immediate: !isCommentMode.value,
  server: !isCommentMode.value
})

const { data: commentData, status: commentStatus } = await useKunFetch<{
  comments: UserGalgameCommentItem[]
  total: number
}>(() => `/user/${props.userId}/galgame-comments`, {
  query: pageData,
  watch: [() => pageData.page, () => pageData.type],
  immediate: isCommentMode.value,
  server: isCommentMode.value
})
</script>

<template>
  <div class="space-y-3">
    <KunTab
      :items="kunUserGalgameNavItem(userId)"
      :model-value="activeTab"
      size="sm"
    />

    <!-- Galgame-card mode (galgame_like / galgame_favorite) -->
    <template v-if="!isCommentMode">
      <div
        v-if="galgameData && galgameData.items.length"
        class="flex flex-col space-y-3"
      >
        <GalgameCard :is-transparent="true" :galgames="galgameData.items" />

        <KunPagination
          v-if="galgameData.total > pageData.limit"
          v-model:current-page="pageData.page"
          :total-page="Math.ceil(galgameData.total / pageData.limit)"
          :is-loading="galgameStatus === 'pending'"
        />
      </div>

      <KunNull
        v-if="galgameData && !galgameData.items.length"
        description="这只笨蛋萝莉没有相关的 Galgame"
      />
    </template>

    <!-- Comment-card mode (galgame_comment / galgame_comment_target / galgame_comment_like) -->
    <template v-else>
      <div
        v-if="commentData && commentData.comments.length"
        class="flex flex-col space-y-3"
      >
        <KunCard
          v-for="c in commentData.comments"
          :key="c.id"
          :href="`/galgame/${c.galgameId}`"
          content-class="space-y-2"
        >
          <KunContent compact :content="renderKatex(c.contentHtml)" />
          <div
            class="text-default-500 flex items-center justify-between text-sm"
          >
            <span>评论于 Galgame #{{ c.galgameId }}</span>
            <KunTime :time="c.created" type="date" show-year />
          </div>
        </KunCard>

        <KunPagination
          v-if="commentData.total > pageData.limit"
          v-model:current-page="pageData.page"
          :total-page="Math.ceil(commentData.total / pageData.limit)"
          :is-loading="commentStatus === 'pending'"
        />
      </div>

      <KunNull
        v-if="commentData && !commentData.comments.length"
        description="这只笨蛋萝莉在 Galgame 下没有相关的评论"
      />
    </template>
  </div>
</template>
