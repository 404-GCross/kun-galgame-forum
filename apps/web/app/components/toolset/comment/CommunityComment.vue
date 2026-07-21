<script setup lang="ts">
// One node in the community-backed toolset comment view. Two flat tiers (grouped
// by the container): depth 0 = root, depth 1 = reply (any DB depth flattened into
// one group). Edit is author-only (parity); delete is author||canModerate (the
// toolset owner is a server-side superset, not a new UI entry — charter ruling 20).
// Like + flag reuse the region-agnostic galgame community components.
const props = withDefaults(
  defineProps<{
    comment: GalgameCommunityComment
    toolsetId: number
    depth?: number
    replies?: GalgameCommunityComment[]
  }>(),
  { depth: 0, replies: () => [] }
)

const emit = defineEmits<{
  replyAdded: [reply: GalgameCommunityComment]
  updated: [post: GalgameCommunityComment]
  tombstoned: [postId: number]
}>()

const { id } = usePersistUserStore()
const canDeleteToolsetComment = useCan('comment.toolset.delete')
const { open: openFlag } = useGalgameCommentFlag()

const isShowReply = ref(false)
const isEditing = ref(false)
const editingContent = ref('')
const isSavingEdit = ref(false)

const isAuthor = computed(() => props.comment.user?.id === id)

const isShowEdit = computed(() => isAuthor.value && !props.comment.deleted)
const isShowDelete = computed(
  () =>
    (isAuthor.value || canDeleteToolsetComment.value) && !props.comment.deleted
)
const isShowFlag = computed(() => !isAuthor.value && !props.comment.deleted)

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
  if (text.length > 1007) {
    useMessage(10541, 'warn')
    return
  }
  if (text === props.comment.content) {
    handleCancelEdit()
    return
  }

  isSavingEdit.value = true
  // Edit reuses the region-agnostic, post-addressed galgame route; no ?gid — the
  // toolset area has no galgame display counter to keep in sync.
  const updated = await kunFetch<GalgameCommunityComment>(
    `/galgame/comments/${props.comment.id}`,
    { method: 'PUT', body: { content: text } }
  )
  isSavingEdit.value = false

  if (updated) {
    useMessage('已更新评论', 'success')
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
  // Region-specific delete: the resource id is pinned by the path so the server
  // decides authority (author / toolset owner / moderator).
  const result = await kunFetch(
    `/toolset/${props.toolsetId}/comments/${props.comment.id}`,
    { method: 'DELETE' }
  )
  if (result) {
    useMessage(10538, 'success')
    emit('tombstoned', props.comment.id)
  }
}

const handleReplyAdded = (reply: GalgameCommunityComment) => {
  isShowReply.value = false
  emit('replyAdded', reply)
}
</script>

<template>
  <div class="flex gap-3">
    <KunAvatar :user="comment.user" :size="depth === 0 ? 'md' : 'sm'" />

    <div class="min-w-0 flex-1 space-y-1.5">
      <div class="flex flex-wrap items-baseline gap-1.5">
        <span class="text-default-800 text-sm font-medium">
          {{ comment.user.name }}
        </span>
        <template v-if="comment.target_user">
          <KunIcon name="lucide:arrow-right" class="text-default-400 text-xs" />
          <KunLink
            underline="hover"
            size="sm"
            :to="`/user/${comment.target_user.id}`"
          >
            {{ comment.target_user.name }}
          </KunLink>
        </template>
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

      <!-- Plain-text view (parity — the toolset area was never markdown). -->
      <div
        v-else-if="!isEditing"
        class="text-default-700 text-sm break-all whitespace-pre-line"
      >
        {{ comment.content }}
      </div>

      <div v-else class="space-y-2">
        <KunTextarea v-model="editingContent" :rows="3" />
        <div class="flex justify-end gap-1">
          <KunButton
            variant="light"
            color="danger"
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
        <div v-if="isShowReply" class="mt-2">
          <ToolsetCommentCommunityComposer
            :toolset-id="toolsetId"
            :reply-to-post-id="comment.id"
            @close="isShowReply = false"
            @submitted="handleReplyAdded"
          />
        </div>
      </KunFadeCard>

      <!-- Replies render flush — single visual tier, smaller avatar the only cue. -->
      <div v-if="depth === 0 && replies.length" class="mt-3 space-y-4">
        <ToolsetCommentCommunityComment
          v-for="reply in replies"
          :key="reply.id"
          :comment="reply"
          :toolset-id="toolsetId"
          :depth="1"
          @reply-added="(r) => emit('replyAdded', r)"
          @updated="(u) => emit('updated', u)"
          @tombstoned="(pid) => emit('tombstoned', pid)"
        />
      </div>
    </div>
  </div>
</template>
