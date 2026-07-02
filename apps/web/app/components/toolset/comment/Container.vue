<script setup lang="ts">
import type { SerializeObject } from 'nitropack'

const props = defineProps<{ toolsetId: number; ownerId?: number }>()

const pageData = reactive({
  toolsetId: props.toolsetId,
  page: 1,
  limit: 30,
  // Roots default oldest-first (先发布在上), consistent with the galgame
  // comment area; changing it re-runs the fetch via the reactive query.
  sort_order: 'asc'
})

const { data, status } = await useKunFetch<{
  comment_data: SerializeObject<ToolsetComment>[]
  total: number
}>(`/toolset/${props.toolsetId}/comment/all`, {
  method: 'GET',
  query: pageData,
  lazy: true
})

// Own the comment list locally so publish / edit / delete are reactive: the
// fetch returns a SHALLOW data ref (Nuxt's default deep:false), so mutating the
// nested data.value.comment_data wouldn't re-render. Re-seed from each fetch
// (sort change / page / refresh re-assign data.value → shallow ref fires).
const comments = ref<SerializeObject<ToolsetComment>[]>([])
const total = ref(0)
watch(
  data,
  (d) => {
    comments.value = d?.comment_data ?? []
    total.value = d?.total ?? 0
  },
  { immediate: true }
)

// `total` counts ROOTS (drives pagination + the empty state), so a new root
// adjusts it but a new reply doesn't.
const addNewComment = (comment: ToolsetComment) => {
  const node = comment as SerializeObject<ToolsetComment>
  if (node.parent_id == null) {
    // Root: end under asc (oldest-first), top under desc.
    if (pageData.sort_order === 'asc') {
      comments.value.push(node)
    } else {
      comments.value.unshift(node)
    }
    total.value++
    return
  }
  // Reply: attach to the root that IS, or CONTAINS, the parent.
  const root = comments.value.find(
    (r) => r.id === node.parent_id || r.reply.some((c) => c.id === node.parent_id)
  )
  if (root) {
    root.reply.push(node)
    root.reply_count = (root.reply_count ?? 0) + 1
  }
}

const removeComment = (commentId: number) => {
  const rootIdx = comments.value.findIndex((c) => c.id === commentId)
  if (rootIdx !== -1) {
    comments.value.splice(rootIdx, 1)
    total.value = Math.max(0, total.value - 1)
    return
  }
  for (const root of comments.value) {
    const idx = root.reply.findIndex((c) => c.id === commentId)
    if (idx !== -1) {
      root.reply.splice(idx, 1)
      root.reply_count = Math.max(0, (root.reply_count ?? 0) - 1)
      return
    }
  }
}

const updateComment = (commentId: number, content: string, edited: string) => {
  for (const root of comments.value) {
    if (root.id === commentId) {
      root.content = content
      root.edited = edited
      return
    }
    const reply = root.reply.find((c) => c.id === commentId)
    if (reply) {
      reply.content = content
      reply.edited = edited
      return
    }
  }
}
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="评论"
      description="如果您对该工具有任何的使用疑问, 欢迎发布评论"
      scale="h2"
    />

    <ToolsetCommentPublish
      :toolset-id="toolsetId"
      @set-new-comment="addNewComment"
    />

    <KunLoading v-if="status === 'pending'" />

    <KunNull v-if="total === 0 && status !== 'pending'" />

    <div class="space-y-3" v-if="total > 0 && status !== 'pending'">
      <div class="flex items-center gap-2">
        <KunButton
          :is-icon-only="true"
          :variant="pageData.sort_order === 'desc' ? 'flat' : 'light'"
          size="lg"
          @click="pageData.sort_order = 'desc'"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-down" />
        </KunButton>

        <KunButton
          :is-icon-only="true"
          :variant="pageData.sort_order === 'asc' ? 'flat' : 'light'"
          size="lg"
          @click="pageData.sort_order = 'asc'"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-up" />
        </KunButton>
      </div>

      <ToolsetComment
        v-for="comment in comments"
        :key="comment.id"
        :comment="comment"
        :toolset-id="toolsetId"
        :owner-id="ownerId || 0"
        @reply-added="addNewComment"
        @reply-edited="updateComment"
        @reply-removed="removeComment"
      />

      <KunPagination
        v-if="total >= pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(total / pageData.limit)"
      />
    </div>
  </div>
</template>
