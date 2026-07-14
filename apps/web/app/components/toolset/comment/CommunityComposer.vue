<script setup lang="ts">
// Compose-box for a community-backed toolset comment (charter step 08).
//
//   - Root mode: no replyToPostId. Container renders it above the thread.
//   - Reply mode: replyToPostId set. Rendered inline beneath the post being
//     replied to (the server derives root/target pointers).
//
// Plain KunTextarea authoring (parity with the legacy ToolsetCommentPublish — the
// toolset area was never markdown).
const props = defineProps<{
  toolsetId: number
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
  const text = content.value.trim()
  if (!text) {
    useMessage(10540, 'warn')
    return
  }
  if (text.length > 1007) {
    useMessage(10541, 'warn')
    return
  }

  isPublishing.value = true
  const result = await kunFetch<GalgameCommunityComment>(
    `/toolset/${props.toolsetId}/comments`,
    {
      method: 'POST',
      body: { content: text, reply_to_post_id: props.replyToPostId ?? null }
    }
  )
  isPublishing.value = false

  if (result) {
    content.value = ''
    useMessage(result.held ? '已提交，待审核' : 10542, 'success')
    emits('submitted', result)
    emits('close')
  }
}
</script>

<template>
  <div class="space-y-2">
    <KunTextarea v-model="content" :rows="3" />
    <div class="flex justify-end">
      <KunButton
        :loading="isPublishing"
        :disabled="!content.trim() || isPublishing"
        @click="handlePublish"
      >
        {{ replyToPostId ? '发布回复' : '发表评论' }}
      </KunButton>
    </div>
  </div>
</template>
