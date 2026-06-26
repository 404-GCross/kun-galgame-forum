<script setup lang="ts">
// One node in the website comment view. Same two-tier flat model as the galgame
// comment area: depth 0 (root) + depth 1 (replies, flattened from any DB depth);
// "A => B" survives via targetUser. A root shows up to 3 replies inline +
// "展开更多 N 条回复" that reveals the rest INLINE (all loaded — no drawer / fetch).
// Website comments have no edit, only reply + delete. Mutations bubble to
// Container, which owns the list.
const props = withDefaults(
  defineProps<{
    comment: WebsiteComment
    websiteId: number
    depth?: number
  }>(),
  { depth: 0 }
)

const emits = defineEmits<{
  replyAdded: [comment: WebsiteComment]
  replyRemoved: [commentId: number]
}>()

const isShowReply = ref(false)
const showAllReplies = ref(false)

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

const onReplyPublished = (comment: WebsiteComment) => {
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
        <template v-if="comment.targetUser">
          <KunIcon name="lucide:arrow-right" class="text-default-400 text-xs" />
          <KunLink
            underline="hover"
            size="sm"
            :to="`/user/${comment.targetUser.id}/info`"
          >
            {{ comment.targetUser.name }}
          </KunLink>
        </template>
        <span class="text-default-400 text-xs">
          <KunTime :time="comment.created" type="datetime" />
        </span>
      </div>

      <p
        :id="`comment-${comment.id}`"
        class="text-default-700 text-sm break-all whitespace-pre-line"
      >
        {{ comment.content }}
      </p>

      <div class="-ml-2 flex items-center gap-1">
        <KunButton
          variant="light"
          size="sm"
          class-name="gap-1"
          @click="isShowReply = !isShowReply"
        >
          <KunIcon name="lucide:reply" />
          回复
        </KunButton>

        <WebsiteCommentDelete
          :comment="comment"
          @remove-comment="(id) => emits('replyRemoved', id)"
        />
      </div>

      <KunFadeCard>
        <div v-if="isShowReply" class="mt-2">
          <WebsiteCommentPublish
            :website-id="websiteId"
            :parent-id="comment.id"
            :receiver="comment.user"
            @on-success="isShowReply = false"
            @set-new-comment="onReplyPublished"
          />
        </div>
      </KunFadeCard>

      <div v-if="visibleReplies.length" class="mt-3 space-y-4">
        <WebsiteCommentRender
          v-for="reply in visibleReplies"
          :key="reply.id"
          :comment="reply"
          :website-id="websiteId"
          :depth="1"
          @reply-added="(c) => emits('replyAdded', c)"
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
