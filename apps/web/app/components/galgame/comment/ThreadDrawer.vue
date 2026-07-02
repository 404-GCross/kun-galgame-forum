<script setup lang="ts">
import type { SerializeObject } from 'nitropack'

// Side drawer that lazy-loads + renders an entire comment thread as a
// flat list (same single-nesting model as the inline view).
//
// The inline list ships only the latest 3 replies per root; the rest
// live behind "查看更多 N 条回复", which opens this drawer and
// fetches `/api/galgame/:gid/comment/thread/:rootId`. The drawer's
// response is the root with ALL descendants flattened into
// root.replies (ASC by created).
//
// In-drawer mutations are applied locally AND forwarded to Container
// so the inline list's replyCount + visible 3 stay in sync.
type Node = SerializeObject<GalgameComment>

const props = defineProps<{
  galgameId: number
  rootCommentId: number | null
}>()

const emit = defineEmits<{
  'update:rootCommentId': [value: number | null]
  replyAdded: [reply: GalgameComment]
  replyEdited: [updated: GalgameComment]
  replyRemoved: [commentId: number, removedSubtreeSize: number, rootId: number]
}>()

const isOpen = computed({
  get: () => props.rootCommentId !== null,
  set: (value) => {
    if (!value) emit('update:rootCommentId', null)
  }
})

const thread = ref<Node | null>(null)
const isLoading = ref(false)

const loadThread = async (rootId: number) => {
  isLoading.value = true
  thread.value = null
  try {
    const data = await kunFetch<Node>(
      `/galgame/${props.galgameId}/comment/thread/${rootId}`,
      { method: 'GET' }
    )
    if (data) thread.value = data
  } finally {
    isLoading.value = false
  }
}

watch(
  () => props.rootCommentId,
  (rootId) => {
    if (rootId !== null) loadThread(rootId)
    else thread.value = null
  }
)

const handleNewComment = (newComment: GalgameComment) => {
  emit('replyAdded', newComment)

  if (!thread.value) return
  const reply = newComment as Node
  if (reply.parent_comment_id == null) return
  if (reply.root_comment_id !== thread.value.id) return

  thread.value = appendReplyToRoot(thread.value, reply)
}

const handleReplyEdited = (updated: GalgameComment) => {
  emit('replyEdited', updated)
  if (!thread.value) return
  const patch = updated as Node
  const { node, replaced } = replaceCommentInRoot(thread.value, patch)
  if (replaced) thread.value = node
}

const handleReplyRemoved = (
  commentId: number,
  removedSize: number,
  rootId: number
) => {
  emit('replyRemoved', commentId, removedSize, rootId)

  if (!thread.value) return

  // Root of the thread itself was deleted → close the drawer; the
  // inline list's Container will splice it out of data.items.
  if (commentId === thread.value.id) {
    emit('update:rootCommentId', null)
    return
  }

  const { node: updated, pruned } = removeReplyFromRoot(
    thread.value,
    commentId,
    removedSize
  )
  if (pruned) thread.value = updated
}
</script>

<template>
  <KunDrawer
    v-model="isOpen"
    placement="right"
    size="lg"
  >
    <template #header>
      <div class="flex flex-col">
        <span class="text-default-800 text-base font-semibold">回复详情</span>
        <span v-if="thread" class="text-default-500 text-xs">
          共 {{ (thread.reply_count ?? 0) + 1 }} 条评论
        </span>
      </div>
    </template>

    <KunLoading v-if="isLoading" />

    <KunNull
      v-else-if="!thread"
      description="评论线程不存在或已被删除"
    />

    <GalgameComment
      v-else
      :comment="thread"
      :depth="0"
      @reply-added="handleNewComment"
      @reply-edited="handleReplyEdited"
      @reply-removed="handleReplyRemoved"
    />
  </KunDrawer>
</template>
