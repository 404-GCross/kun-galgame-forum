<script setup lang="ts">
// One node in the community-backed comment view. The visual model stays FLAT —
// two tiers:
//
//   depth 0 = root; depth 1 = reply (any DB depth, rendered as one flat group)
//
// The container does the grouping and passes a root's replies down via `replies`;
// a reply node never receives children. Mutations bubble up as events; the
// container owns the list state.
const props = withDefaults(
  defineProps<{
    comment: GalgameCommunityComment
    galgameId: number
    depth?: number
    // Root-only: the flat reply group beneath this post (grouped by the
    // container). Empty / absent on a reply node.
    replies?: GalgameCommunityComment[]
  }>(),
  { depth: 0, replies: () => [] }
)

const emit = defineEmits<{
  replyAdded: [reply: GalgameCommunityComment]
  // Author/moderator edit success — the patched node, spliced in by id.
  updated: [post: GalgameCommunityComment]
  // Delete tombstones the post in place (the floor is kept); the container flips
  // its `deleted` flag rather than removing the row.
  tombstoned: [postId: number]
}>()

const { id } = usePersistUserStore()
const { canModerate } = useRole()
const { open: openFlag } = useGalgameCommentFlag()

const isShowReply = ref(false)
const isEditing = ref(false)
const editingContent = ref('')
const isSavingEdit = ref(false)

const isAuthor = computed(() => props.comment.user?.id === id)

// Authority is aligned with the server: only the author edits; the author OR a
// moderator deletes. The legacy UI also showed delete to the galgame owner, but
// the server never allowed it — so the owner no longer sees the button.
const isShowEdit = computed(() => isAuthor.value && !props.comment.deleted)
const isShowDelete = computed(
  () => (isAuthor.value || canModerate.value) && !props.comment.deleted
)
// Report entry: not on your own post, and never on a tombstone.
const isShowFlag = computed(() => !isAuthor.value && !props.comment.deleted)

// "已编辑" / "已编辑（管理）" — a moderator rewrite wins the label.
const editedLabel = computed(() => {
  if (props.comment.edited == null && !props.comment.edited_by_moderator) {
    return null
  }
  return props.comment.edited_by_moderator ? '已编辑（管理）' : '已编辑'
})

const handleFlag = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  openFlag(props.comment.id)
}

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
  // PUT/DELETE carry ?gid so the BFF keeps the local display counter + mention
  // deep-links (the community response has no galgame anchor of its own).
  const updated = await kunFetch<GalgameCommunityComment>(
    `/galgame/comments/${props.comment.id}`,
    { method: 'PUT', query: { gid: props.galgameId }, body: { content: text } }
  )
  isSavingEdit.value = false

  if (updated) {
    useMessage('评论已更新', 'success')
    emit('updated', updated)
    handleCancelEdit()
  }
}

const handleDelete = async () => {
  const ok = await useComponentMessageStore().alert(
    '删除后此楼保留占位，回复不受影响，确定删除吗？'
  )
  if (!ok) {
    return
  }

  const result = await kunFetch(`/galgame/comments/${props.comment.id}`, {
    method: 'DELETE',
    query: { gid: props.galgameId }
  })

  if (result) {
    useMessage(10538, 'success')
    emit('tombstoned', props.comment.id)
  }
}
</script>

<template>
  <!-- Anchor for notification / external deep-links (?comment=<post_id> and
       #galgame-comment-<post_id>); the container scrolls to + highlights it. -->
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
        <span v-if="editedLabel" class="text-default-400 text-xs italic">
          ({{ editedLabel }})
        </span>
        <span
          v-if="comment.held"
          class="bg-warning-100 text-warning-600 rounded px-1.5 py-0.5 text-xs"
        >
          审核中
        </span>
      </div>

      <!-- Tombstone: keep the floor, gray the body. -->
      <p v-if="comment.deleted" class="text-default-400 text-sm italic">
        [已删除]
      </p>

      <!-- View mode: kungal-rendered Markdown (KunContent → DOMPurify). -->
      <KunContent
        v-else-if="!isEditing"
        compact
        :content="renderKatex(comment.content_html)"
      />

      <!-- Edit mode: same editor as the composer, pre-loaded with the source. -->
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
          <KunButton size="sm" :loading="isSavingEdit" @click="handleSubmitEdit">
            保存
          </KunButton>
        </div>
      </div>

      <div
        v-if="!comment.deleted && !isEditing"
        class="-ml-2 flex items-center gap-1"
      >
        <KunButton
          variant="light"
          size="sm"
          class-name="gap-1"
          @click="isShowReply = !isShowReply"
        >
          <KunIcon name="lucide:reply" />
          回复
        </KunButton>

        <GalgameCommentCommunityLike :comment="comment" />

        <KunTooltip v-if="isShowFlag" text="举报">
          <KunButton
            :is-icon-only="true"
            color="danger"
            variant="light"
            size="sm"
            @click="handleFlag"
          >
            <KunIcon name="lucide:flag" />
          </KunButton>
        </KunTooltip>

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
        <GalgameCommentCommunityComposer
          v-if="isShowReply"
          :galgame-id="galgameId"
          :reply-to-post-id="comment.id"
          @close="isShowReply = false"
          @submitted="(reply) => emit('replyAdded', reply)"
        />
      </KunFadeCard>

      <!-- Replies render flush — single visual tier, smaller avatar the only cue. -->
      <div v-if="depth === 0 && replies.length" class="mt-3 space-y-4">
        <GalgameCommentCommunityComment
          v-for="reply in replies"
          :key="reply.id"
          :comment="reply"
          :galgame-id="galgameId"
          :depth="1"
          @reply-added="(r) => emit('replyAdded', r)"
          @updated="(u) => emit('updated', u)"
          @tombstoned="(pid) => emit('tombstoned', pid)"
        />
      </div>
    </div>
  </div>
</template>
