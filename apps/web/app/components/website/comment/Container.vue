<script setup lang="ts">
import { scrollIntoComment } from './_scrollIntoComment'

const props = defineProps<{
  websiteId: number
}>()

const { data } = await useKunFetch<WebsiteComment[]>(
  `/website/${props.websiteId}/comment`,
  {
    watch: false,
    query: { website_id: props.websiteId }
  }
)

// Roots (each carrying its replies flattened oldest-first). Owned locally so
// publish / delete stay reactive (the fetch data ref is shallow).
const comments = ref<WebsiteComment[]>(data.value ?? [])

const addNewComment = (newComment: WebsiteComment) => {
  if (newComment.parent_id == null) {
    // Root: append (the list is oldest-first) and scroll to it.
    comments.value.push(newComment)
    nextTick(() => scrollIntoComment(newComment.id))
    return
  }
  // Reply: attach to the root that IS, or CONTAINS, the parent.
  const root = comments.value.find(
    (r) =>
      r.id === newComment.parent_id ||
      r.reply.some((c) => c.id === newComment.parent_id)
  )
  if (root) {
    root.reply.push(newComment)
    root.reply_count = (root.reply_count ?? 0) + 1
  }
}

const removeComment = (commentId: number) => {
  const rootIdx = comments.value.findIndex((c) => c.id === commentId)
  if (rootIdx !== -1) {
    comments.value.splice(rootIdx, 1)
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
</script>

<template>
  <KunCard :is-transparent="false" :is-hoverable="false" class-name="p-6">
    <h2 class="text-default-900 mb-6 text-2xl font-bold">用户评论</h2>

    <div class="mb-8">
      <WebsiteCommentPublish
        :website-id="websiteId"
        @set-new-comment="addNewComment"
      />
    </div>

    <div v-if="comments.length" class="space-y-6">
      <WebsiteCommentRender
        v-for="comment in comments"
        :key="comment.id"
        :comment="comment"
        :website-id="websiteId"
        @reply-added="addNewComment"
        @reply-removed="removeComment"
      />
    </div>
  </KunCard>
</template>
