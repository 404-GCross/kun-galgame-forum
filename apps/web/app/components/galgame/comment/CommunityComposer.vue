<script setup lang="ts">
// Compose-box for a community-backed comment.
//
//   - Root mode: no replyToPostId. Container renders it above the thread.
//   - Reply mode: replyToPostId set. CommunityComment renders it inline beneath
//     the post being replied to (the server derives root/target pointers).
//
// Same KunMilkdownDualEditorProvider as the legacy Panel, so the authoring
// experience is identical. The wire payload is raw Markdown.
const props = defineProps<{
  galgameId: number
  replyToPostId?: number | null
}>()

const emits = defineEmits<{
  close: []
  // Optimistic update: the server-returned post so the parent can splice it into
  // the list without re-fetching. A held (TL0) post still comes back so its
  // author sees their own "审核中" comment.
  submitted: [post: GalgameCommunityComment]
}>()

const content = ref('')
const isPublishing = ref(false)

const handlePublish = async () => {
  const trimmed = content.value.trim()
  if (!trimmed) {
    useMessage(10540, 'warn')
    return
  }
  if (trimmed.length > 5000) {
    useMessage(10541, 'warn')
    return
  }

  isPublishing.value = true
  const result = await kunFetch<GalgameCommunityComment>(
    `/galgame/${props.galgameId}/comments`,
    {
      method: 'POST',
      body: {
        content: content.value,
        reply_to_post_id: props.replyToPostId ?? null
      }
    }
  )
  isPublishing.value = false

  if (result) {
    content.value = ''
    // A TL0 first post is held for review; the receipt says so, but we still
    // insert it locally (the author sees their own held post).
    useMessage(result.held ? '已提交，待审核' : 10542, 'success')
    emits('submitted', result)
    emits('close')
  }
}
</script>

<template>
  <div class="space-y-3">
    <KunMilkdownDualEditorProvider
      :value-markdown="content"
      placeholder="请温柔的发表你的看法吧～「评论给」已废除，@用户名 即可通知对方"
      @set-markdown="(val) => (content = val)"
    />

    <div class="flex items-center justify-between gap-2">
      <slot />

      <KunButton class="ml-auto" :loading="isPublishing" @click="handlePublish">
        {{ replyToPostId ? '发布回复' : '发布评论' }}
      </KunButton>
    </div>
  </div>
</template>
