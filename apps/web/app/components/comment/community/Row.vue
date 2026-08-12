<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    comment: GalgameCommunityComment
    target: CommunityCommentTarget
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

const surface = communityCommentSurface(props.target)

const { id } = usePersistUserStore()
const canDeleteComment = useCan(surface.deletePermission)
const { open: openFlag } = useGalgameCommentFlag()

const isShowReply = ref(false)
const isEditing = ref(false)
const editingContent = ref('')
const isSavingEdit = ref(false)

const isAuthor = computed(() => props.comment.user?.id === id)

const isShowEdit = computed(() => isAuthor.value && !props.comment.deleted)
const isShowDelete = computed(
  () => (isAuthor.value || canDeleteComment.value) && !props.comment.deleted
)
const isShowFlag = computed(() => !isAuthor.value && !props.comment.deleted)

const isShowMenu = computed(
  () => isShowEdit.value || isShowFlag.value || isShowDelete.value
)

const replyTarget = computed(() =>
  surface.showsReplyTarget ? props.comment.target_user : null
)

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
  if (text.length > surface.maxLength) {
    useMessage(`评论最大长度为 ${surface.maxLength} 个字符`, 'warn')
    return
  }
  if (text === props.comment.content) {
    handleCancelEdit()
    return
  }

  isSavingEdit.value = true
  const updated = await kunFetch<GalgameCommunityComment>(
    surface.editUrl(props.comment.id),
    {
      method: 'PUT',
      query: surface.editQuery,
      body: { content: text }
    }
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

  const result = await kunFetch(surface.deleteUrl(props.comment.id), {
    method: 'DELETE',
    query: surface.deleteQuery
  })

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
  <div :id="`${surface.anchorPrefix}-${comment.id}`" class="flex gap-3">
    <KunAvatar :user="comment.user" :size="depth === 0 ? 'md' : 'sm'" />

    <div class="min-w-0 flex-1">
      <div
        class="flex flex-wrap items-baseline gap-x-2 gap-y-1 text-xs leading-5"
      >
        <span class="text-default-800 text-sm font-medium">
          {{ comment.user.name }}
        </span>
        <template v-if="replyTarget">
          <KunIcon name="lucide:arrow-right" class="text-default-400" />
          <KunLink underline="hover" size="sm" :to="`/user/${replyTarget.id}`">
            {{ replyTarget.name }}
          </KunLink>
        </template>
        <span class="text-default-400">
          <KunTime :time="comment.created" />
        </span>
        <span v-if="editedLabel" class="text-default-400 italic">
          {{ editedLabel }}
        </span>
        <span
          v-if="comment.held"
          class="bg-warning-100 text-warning-600 rounded-full px-2 py-0.5"
        >
          审核中
        </span>
      </div>

      <p v-if="comment.deleted" class="text-default-400 mt-2 text-sm italic">
        [已删除]
      </p>

      <KunContent
        v-else-if="!isEditing"
        class="mt-2"
        compact
        :content="renderKatex(comment.content_html)"
      />

      <div v-else class="mt-2 space-y-2">
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

      <div
        v-if="!comment.deleted && !isEditing"
        class="mt-2.5 flex items-center gap-1"
      >
        <KunTooltip text="回复">
          <KunReaction
            :toggle="false"
            icon="lucide:reply"
            label="回复"
            @click="isShowReply = !isShowReply"
          />
        </KunTooltip>

        <CommentCommunityLike :comment="comment" />

        <KunPopover v-if="isShowMenu" position="bottom-start">
          <template #trigger>
            <KunReaction :toggle="false" icon="lucide:ellipsis" label="更多" />
          </template>

          <div class="flex w-44 flex-col gap-2 p-2">
            <KunButton
              v-if="isShowEdit"
              variant="light"
              color="default"
              size="sm"
              class-name="w-full justify-start gap-2 whitespace-nowrap"
              @click="handleStartEdit"
            >
              <KunIcon class-name="text-lg" name="lucide:pencil" />
              编辑评论
            </KunButton>

            <KunButton
              v-if="isShowFlag"
              variant="light"
              color="danger"
              size="sm"
              class-name="w-full justify-start gap-2 whitespace-nowrap"
              @click="handleFlag"
            >
              <KunIcon class-name="text-lg" name="lucide:flag" />
              举报评论
            </KunButton>

            <KunButton
              v-if="isShowDelete"
              variant="light"
              color="danger"
              size="sm"
              class-name="w-full justify-start gap-2 whitespace-nowrap"
              @click="handleDelete"
            >
              <KunIcon class-name="text-lg" name="lucide:trash-2" />
              删除评论
            </KunButton>
          </div>
        </KunPopover>
      </div>

      <KunFadeCard>
        <CommentCommunityComposer
          v-if="isShowReply"
          class="mt-3"
          :target="target"
          :reply-to-post-id="surface.isFlat ? null : comment.id"
          :target-user-id="comment.user.id"
          :is-reply="true"
          @close="isShowReply = false"
          @submitted="handleReplyAdded"
        />
      </KunFadeCard>

      <div
        v-if="!surface.isFlat && depth === 0 && replies.length"
        class="mt-4 space-y-4"
      >
        <CommentCommunityRow
          v-for="reply in replies"
          :key="reply.id"
          :comment="reply"
          :target="target"
          :depth="1"
          @reply-added="(r) => emit('replyAdded', r)"
          @updated="(u) => emit('updated', u)"
          @tombstoned="(pid) => emit('tombstoned', pid)"
        />
      </div>
    </div>
  </div>
</template>
