<script setup lang="ts">
// One node in the toolset comment view. Visual model mirrors the galgame
// comment area: two flat tiers — depth 0 (root) and depth 1 (reply). Replies of
// any DB depth are flattened under the root (the "A => B" context survives via
// targetUser). A root shows up to 3 replies inline + a "展开更多 N 条回复" toggle
// that reveals the rest INLINE — all replies ship in the payload, so there's no
// drawer and no lazy fetch (the deliberate difference from galgame). Mutations
// bubble to Container, which owns the list.
const props = withDefaults(
  defineProps<{
    comment: ToolsetComment
    toolsetId: number
    ownerId: number
    depth?: number
  }>(),
  { depth: 0 }
)

const emits = defineEmits<{
  replyAdded: [comment: ToolsetComment]
  replyEdited: [id: number, content: string, edited: string]
  replyRemoved: [id: number]
}>()

const { id: myId } = usePersistUserStore()
const { canModerate } = useRole()

const isShowReply = ref(false)
const isEditing = ref(false)
const editContent = ref('')
const isSaving = ref(false)
const showAllReplies = ref(false)

const canDelete = computed(
  () =>
    props.comment.user.id === myId ||
    props.ownerId === myId ||
    canModerate.value
)
const canEdit = computed(() => props.comment.user.id === myId)
const isEdited = computed(() => props.comment.edited != null)

// Only roots show children; replies render leaf-only. Show 3 until expanded.
const visibleReplies = computed(() => {
  if (props.depth !== 0) {
    return []
  }
  const all = props.comment.reply ?? []
  return showAllReplies.value ? all : all.slice(0, 3)
})
const hiddenReplyCount = computed(
  () => (props.comment.reply?.length ?? 0) - visibleReplies.value.length
)

const handleDelete = async () => {
  const ok = await useComponentMessageStore().alert('确认删除该评论？')
  if (!ok) {
    return
  }
  const res = await kunFetch(`/toolset/${props.toolsetId}/comment`, {
    method: 'DELETE',
    query: { comment_id: props.comment.id }
  })
  if (res) {
    useMessage(10538, 'success')
    emits('replyRemoved', props.comment.id)
  }
}

const startEdit = () => {
  isEditing.value = true
  editContent.value = props.comment.content
}
const cancelEdit = () => {
  isEditing.value = false
  editContent.value = ''
}
const saveEdit = async () => {
  if (!editContent.value.trim()) {
    useMessage(10540, 'warn')
    return
  }
  isSaving.value = true
  const ok = await kunFetch(`/toolset/${props.toolsetId}/comment`, {
    method: 'PUT',
    body: { comment_id: props.comment.id, content: editContent.value }
  })
  isSaving.value = false
  if (ok) {
    emits(
      'replyEdited',
      props.comment.id,
      editContent.value,
      new Date().toISOString()
    )
    useMessage('已更新评论', 'success')
    cancelEdit()
  }
}

const onReplyPublished = (comment: ToolsetComment) => {
  isShowReply.value = false
  emits('replyAdded', comment)
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
        <span v-if="isEdited" class="text-default-400 text-xs italic">
          (已编辑)
        </span>
      </div>

      <div
        v-if="!isEditing"
        class="text-default-700 text-sm break-all whitespace-pre-line"
      >
        {{ comment.content }}
      </div>
      <div v-else class="space-y-2">
        <KunTextarea v-model="editContent" :rows="3" />
        <div class="flex justify-end gap-1">
          <KunButton
            variant="light"
            color="danger"
            size="sm"
            @click="cancelEdit"
          >
            取消
          </KunButton>
          <KunButton size="sm" :loading="isSaving" @click="saveEdit">
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

        <KunTooltip v-if="canEdit" text="编辑">
          <KunButton
            :is-icon-only="true"
            variant="light"
            size="sm"
            @click="startEdit"
          >
            <KunIcon name="lucide:pencil" />
          </KunButton>
        </KunTooltip>

        <KunTooltip v-if="canDelete" text="删除">
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
          <ToolsetCommentPublish
            :toolset-id="toolsetId"
            :parent-id="comment.id"
            @set-new-comment="onReplyPublished"
          />
        </div>
      </KunFadeCard>

      <!-- Replies render flush as one flat tier; the smaller avatar is the
           only hierarchy cue (mirrors the galgame comment area). -->
      <div v-if="visibleReplies.length" class="mt-3 space-y-4">
        <ToolsetComment
          v-for="reply in visibleReplies"
          :key="reply.id"
          :comment="reply"
          :toolset-id="toolsetId"
          :owner-id="ownerId"
          :depth="1"
          @reply-added="(c) => emits('replyAdded', c)"
          @reply-edited="
            (id, content, edited) => emits('replyEdited', id, content, edited)
          "
          @reply-removed="(id) => emits('replyRemoved', id)"
        />
      </div>

      <KunButton
        v-if="depth === 0 && hiddenReplyCount > 0"
        variant="light"
        color="primary"
        size="sm"
        full-width
        class-name="mt-2"
        @click="showAllReplies = true"
      >
        <KunIcon name="lucide:messages-square" />
        展开更多 {{ hiddenReplyCount }} 条回复
      </KunButton>
    </div>
  </div>
</template>
