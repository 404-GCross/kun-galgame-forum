<script setup lang="ts">
// One node in the community-backed rating comment view. The rating area is FLAT:
// there is no grouping — every comment (including a "reply") is a root post that
// simply carries a different target_user ("A → B"). Replying just opens a composer
// pre-targeted at THIS comment's author; the new post lands flat in the container's
// list. Edit is author-only (parity); delete is author||canModerate (the rated
// galgame's owner is a server-side superset, not a new UI entry — charter ruling
// 20). Like + flag reuse the region-agnostic galgame community components.
const props = defineProps<{
  comment: GalgameCommunityComment
  ratingId: number
}>()

const emit = defineEmits<{
  replyAdded: [reply: GalgameCommunityComment]
  updated: [post: GalgameCommunityComment]
  tombstoned: [postId: number]
}>()

const { id } = usePersistUserStore()
const canDeleteRatingComment = useCan('comment.rating.delete')
const { open: openFlag } = useGalgameCommentFlag()

const isShowReply = ref(false)
const isEditing = ref(false)
const editingContent = ref('')
const isSavingEdit = ref(false)

const isAuthor = computed(() => props.comment.user?.id === id)

const isShowEdit = computed(() => isAuthor.value && !props.comment.deleted)
const isShowDelete = computed(
  () =>
    (isAuthor.value || canDeleteRatingComment.value) && !props.comment.deleted
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
    useMessage('请输入评论内容', 'warn')
    return
  }
  if (text.length > 1314) {
    useMessage('内容长度不能超过 1314 字', 'warn')
    return
  }
  if (text === props.comment.content) {
    handleCancelEdit()
    return
  }

  isSavingEdit.value = true
  // Edit reuses the region-agnostic, post-addressed galgame route; no ?gid — the
  // rating area has no galgame display counter to keep in sync.
  const updated = await kunFetch<GalgameCommunityComment>(
    `/galgame/comments/${props.comment.id}`,
    { method: 'PUT', body: { content: text } }
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
  // Region-specific delete: the resource id is pinned by the path so the server
  // decides authority (author / rated-galgame owner / moderator).
  const result = await kunFetch(
    `/galgame-rating/${props.ratingId}/comments/${props.comment.id}`,
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
  <KunCard :is-hoverable="false" content-class="flex-row gap-3">
    <KunAvatar :user="comment.user" />

    <div class="flex w-full min-w-0 flex-col space-y-2">
      <div class="flex flex-wrap items-center gap-1.5">
        <span class="text-default-700">{{ comment.user.name }}</span>
        <template v-if="comment.target_user">
          <span>=></span>
          <KunLink underline="hover" :to="`/user/${comment.target_user.id}`">
            {{ comment.target_user.name }}
          </KunLink>
        </template>
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

      <!-- Plain-text view (parity — the rating area was never markdown). -->
      <div v-else-if="!isEditing" class="break-all whitespace-pre-line">
        {{ comment.content }}
      </div>

      <div v-else class="space-y-2">
        <KunTextarea v-model="editingContent" auto-grow />
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
        class="flex items-end justify-between"
      >
        <span class="text-default-500 text-sm">
          发布于 <KunTime :time="comment.created" />
        </span>

        <div class="flex items-center justify-end gap-1">
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
      </div>

      <KunFadeCard>
        <GalgameRatingCommentCommunityComposer
          v-if="isShowReply"
          :rating-id="ratingId"
          :target-user-id="comment.user.id"
          @close="isShowReply = false"
          @submitted="handleReplyAdded"
        />
      </KunFadeCard>
    </div>
  </KunCard>
</template>
