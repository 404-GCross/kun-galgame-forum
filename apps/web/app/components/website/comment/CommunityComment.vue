<script setup lang="ts">
// One node in the community-backed website comment view. Two flat tiers (grouped
// by the container): depth 0 = root, depth 1 = reply (any DB depth flattened into
// one group). Website comments have NO edit (parity — the old website area never
// had an edit route, and the new backend keeps it dormant); only reply + delete
// (author||canModerate) + like + flag. Mutations bubble up; the container owns the
// list. The row carries a #website-comment-<id> anchor the container scrolls to
// after a publish (parity with the legacy _scrollIntoComment).
const props = withDefaults(
  defineProps<{
    comment: GalgameCommunityComment
    websiteId: number
    domain: string
    depth?: number
    replies?: GalgameCommunityComment[]
  }>(),
  { depth: 0, replies: () => [] }
)

const emit = defineEmits<{
  replyAdded: [reply: GalgameCommunityComment]
  tombstoned: [postId: number]
}>()

const { id } = usePersistUserStore()
const canDeleteWebsiteComment = useCan('comment.website.delete')
const { open: openFlag } = useGalgameCommentFlag()

const isShowReply = ref(false)

const isAuthor = computed(() => props.comment.user?.id === id)
const isShowDelete = computed(
  () =>
    (isAuthor.value || canDeleteWebsiteComment.value) && !props.comment.deleted
)
const isShowFlag = computed(() => !isAuthor.value && !props.comment.deleted)

const handleFlag = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  openFlag(props.comment.id)
}

const handleDelete = async () => {
  const ok = await useComponentMessageStore().alert(
    '你这个坏萝莉, 确定删除这个评论吗?',
    '删除后此楼保留占位，回复不受影响'
  )
  if (!ok) {
    return
  }
  // Region-specific delete: website_id addresses the resource (:domain decorative).
  const result = await kunFetch(
    `/website/${props.domain}/comments/${props.comment.id}`,
    { method: 'DELETE', query: { website_id: props.websiteId } }
  )
  if (result) {
    useMessage('删除评论成功', 'success')
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
          <KunTime :time="comment.created" type="datetime" />
        </span>
        <span
          v-if="comment.held"
          class="bg-warning-100 text-warning-600 rounded px-1.5 py-0.5 text-xs"
        >
          审核中
        </span>
      </div>

      <!-- Tombstone: keep the floor, gray the body. -->
      <p
        v-if="comment.deleted"
        :id="`website-comment-${comment.id}`"
        class="text-default-400 text-sm italic"
      >
        [已删除]
      </p>

      <!-- Plain-text view (parity — the website area was never markdown). -->
      <p
        v-else
        :id="`website-comment-${comment.id}`"
        class="text-default-700 text-sm break-all whitespace-pre-line"
      >
        {{ comment.content }}
      </p>

      <div v-if="!comment.deleted" class="-ml-2 flex items-center gap-1">
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
          <WebsiteCommentCommunityComposer
            :website-id="websiteId"
            :domain="domain"
            :reply-to-post-id="comment.id"
            @close="isShowReply = false"
            @submitted="handleReplyAdded"
          />
        </div>
      </KunFadeCard>

      <!-- Replies render flush — single visual tier, smaller avatar the only cue. -->
      <div v-if="depth === 0 && replies.length" class="mt-3 space-y-4">
        <WebsiteCommentCommunityComment
          v-for="reply in replies"
          :key="reply.id"
          :comment="reply"
          :website-id="websiteId"
          :domain="domain"
          :depth="1"
          @reply-added="(r) => emit('replyAdded', r)"
          @tombstoned="(pid) => emit('tombstoned', pid)"
        />
      </div>
    </div>
  </div>
</template>
