<script setup lang="ts">
import type { SerializeObject } from 'nitropack'

// One node in the comment view. The visual model is FLAT — two tiers:
//
//   depth 0 = root (top-level comment)
//   depth 1 = reply (anything in the root's flat replies list)
//
// We never render past depth 1. The DB still tracks the true parent
// chain via parent_comment_id, but the inline list and the drawer
// both render replies as one flat group beneath the root. To notify the
// person you're replying to, @-mention them in the comment body.
//
// Mutations bubble up as events; Container / ThreadDrawer own the
// state and apply immutable updates.
const props = withDefaults(
  defineProps<{
    comment: SerializeObject<GalgameComment>
    depth?: number
  }>(),
  { depth: 0 }
)

const emit = defineEmits<{
  openThread: [rootId: number]
  replyAdded: [reply: GalgameComment]
  // Author-edit success carries the patched node so the controller
  // can splice it in by id without re-fetching.
  replyEdited: [updated: GalgameComment]
  // Carries rootId so the controller can locate the affected root
  // even when the deleted comment isn't in the loaded slice.
  replyRemoved: [commentId: number, removedSubtreeSize: number, rootId: number]
}>()

const galgame = inject<GalgameDetail>('galgame')
const { id } = usePersistUserStore()
const { canModerate } = useRole()

const isShowReply = ref(false)
const isEditing = ref(false)
const editingContent = ref('')
const isSavingEdit = ref(false)

const isShowDelete = computed(
  () =>
    props.comment.user?.id === id ||
    galgame?.user.id === id ||
    canModerate.value
)
// Only the author can edit. Moderators can delete but not silently
// rewrite — that's a deliberate authority boundary.
const isShowEdit = computed(() => props.comment.user?.id === id)

const isEdited = computed(() => props.comment.edited != null)

// Only the root has visible children; replies render leaf-only.
const visibleReplies = computed(() =>
  props.depth === 0 ? (props.comment.replies ?? []) : []
)

// "查看更多 N 条回复" — only ever appears under a root that has
// more total descendants than what's been loaded into .replies.
const showLoadMore = computed(
  () =>
    props.depth === 0 &&
    (props.comment.reply_count ?? 0) > visibleReplies.value.length
)

const rootAnchorId = computed(
  () => props.comment.root_comment_id ?? props.comment.id
)

const handleStartEdit = () => {
  editingContent.value = props.comment.content
  isEditing.value = true
}

const handleCancelEdit = () => {
  isEditing.value = false
  editingContent.value = ''
}

const handleSubmitEdit = async () => {
  const text = editingContent.value.trim()
  if (!text) {
    useMessage(10540, 'warn')
    return
  }
  if (text.length > 5000) {
    useMessage(10541, 'warn')
    return
  }
  if (text === props.comment.content) {
    handleCancelEdit()
    return
  }

  isSavingEdit.value = true
  const updated = await kunFetch<GalgameComment>(
    `/galgame/${props.comment.galgame_id}/comment`,
    {
      method: 'PUT',
      body: { comment_id: props.comment.id, content: text }
    }
  )
  isSavingEdit.value = false

  if (updated) {
    useMessage('评论已更新', 'success')
    emit('replyEdited', updated)
    handleCancelEdit()
  }
}

const handleDelete = async () => {
  const ok = await useComponentMessageStore().alert('您确定删除评论吗？')
  if (!ok) return

  const result = await kunFetch(`/galgame/${props.comment.galgame_id}/comment`, {
    method: 'DELETE',
    query: { comment_id: props.comment.id }
  })

  if (result) {
    useMessage(10538, 'success')
    const removed = 1 + (props.comment.reply_count ?? 0)
    emit('replyRemoved', props.comment.id, removed, rootAnchorId.value)
  }
}
</script>

<template>
  <!-- Anchor for notification deep-links (?comment=<id>); scrolled to +
       highlighted by the thread drawer. -->
  <div :id="`galgame-comment-${comment.id}`" class="flex gap-3">
    <KunAvatar :user="comment.user" :size="depth === 0 ? 'md' : 'sm'" />

    <div class="min-w-0 flex-1 space-y-1.5">
      <div class="flex flex-wrap items-baseline gap-1.5">
        <span class="text-default-800 text-sm font-medium">
          {{ comment.user.name }}
        </span>
        <span class="text-default-400 text-xs">
          <KunTime :time="comment.created" />
        </span>
        <span v-if="isEdited" class="text-default-400 text-xs italic">
          (已编辑)
        </span>
      </div>

      <!-- View mode: server-rendered Markdown (KunContent sanitizes
           via DOMPurify and applies project-wide kun-prose styling). -->
      <KunContent
        v-if="!isEditing"
        compact
        :content="renderKatex(comment.content_html)"
      />

      <!-- Edit mode: same Milkdown editor as the new-comment Panel,
           pre-loaded with the existing markdown source. -->
      <div v-else class="space-y-2">
        <KunMilkdownDualEditorProvider
          :value-markdown="editingContent"
          @set-markdown="(val) => (editingContent = val)"
        />
        <div class="flex justify-end gap-2">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            @click="handleCancelEdit"
          >
            取消
          </KunButton>
          <KunButton
            size="sm"
            :loading="isSavingEdit"
            @click="handleSubmitEdit"
          >
            保存
          </KunButton>
        </div>
      </div>

      <div v-if="!isEditing" class="-ml-2 flex items-center gap-1">
        <KunButton
          variant="light"
          size="sm"
          class-name="gap-1"
          @click="isShowReply = !isShowReply"
        >
          <KunIcon name="lucide:reply" />
          回复
        </KunButton>

        <GalgameCommentLike :comment="comment" />

        <KunTooltip v-if="isShowEdit" text="编辑">
          <KunButton
            :is-icon-only="true"
            variant="light"
            color="default"
            size="sm"
            @click="handleStartEdit"
          >
            <KunIcon name="lucide:pencil" />
          </KunButton>
        </KunTooltip>

        <KunTooltip v-if="isShowDelete" text="删除">
          <KunButton
            :is-icon-only="true"
            variant="light"
            color="danger"
            size="sm"
            @click="handleDelete"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </KunTooltip>
      </div>

      <KunFadeCard>
        <GalgameCommentPanel
          v-if="isShowReply"
          :parent-comment-id="comment.id"
          @close="isShowReply = false"
          @submitted="(reply) => emit('replyAdded', reply)"
        />
      </KunFadeCard>

      <!-- Replies render flush — single visual tier, no indent or
           border-l. Smaller avatar on the children is the only
           hierarchy cue. -->
      <div v-if="visibleReplies.length" class="mt-3 space-y-4">
        <GalgameComment
          v-for="reply in visibleReplies"
          :key="reply.id"
          :comment="reply"
          :depth="1"
          @reply-added="(r) => emit('replyAdded', r)"
          @reply-edited="(u) => emit('replyEdited', u)"
          @reply-removed="
            (id, size, rootId) => emit('replyRemoved', id, size, rootId)
          "
        />
      </div>

      <KunButton
        v-if="showLoadMore"
        variant="light"
        color="primary"
        size="sm"
        full-width
        class-name="mt-2"
        @click="emit('openThread', rootAnchorId)"
      >
        <KunIcon name="lucide:messages-square" />
        查看更多 {{ comment.reply_count }} 条回复
      </KunButton>
    </div>
  </div>
</template>
