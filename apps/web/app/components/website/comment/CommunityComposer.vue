<script setup lang="ts">
// Compose-box for a community-backed website comment (charter step 08).
//
//   - Root mode: no replyToPostId. Container renders it above the thread.
//   - Reply mode: replyToPostId set. Rendered inline beneath the post being
//     replied to (the server derives root/target pointers).
//
// Plain KunTextarea authoring (parity with the legacy WebsiteCommentPublish — the
// website area was never markdown). The addressing website_id rides the body; the
// :domain path segment is decorative (the API keys on the numeric id).
const props = defineProps<{
  websiteId: number
  domain: string
  replyToPostId?: number | null
}>()

const emits = defineEmits<{
  close: []
  // Optimistic update: the server-returned post so the parent can splice it into
  // the list without re-fetching. A held (TL0) post still comes back so its
  // author sees their own "审核中" comment.
  submitted: [post: GalgameCommunityComment]
}>()

const userStore = usePersistUserStore()

const content = ref('')
const isPublishing = ref(false)

const handlePublish = async () => {
  if (!userStore.id) {
    useAuthModal().open()
    return
  }
  const text = content.value.trim()
  if (!text) {
    useMessage('评论内容不能为空', 'warn')
    return
  }
  if (text.length > 1007) {
    useMessage('内容长度不能超过 1007 字', 'warn')
    return
  }

  isPublishing.value = true
  const result = await kunFetch<GalgameCommunityComment>(
    `/website/${props.domain}/comments`,
    {
      method: 'POST',
      body: {
        content: text,
        website_id: props.websiteId,
        reply_to_post_id: props.replyToPostId ?? null
      }
    }
  )
  isPublishing.value = false

  if (result) {
    content.value = ''
    useMessage(result.held ? '已提交，待审核' : '发布评论成功', 'success')
    emits('submitted', result)
    emits('close')
  }
}
</script>

<template>
  <div class="flex space-x-3">
    <KunAvatar
      :user="{ id: userStore.id, name: userStore.name, avatar: userStore.avatar }"
      :disable-floating="true"
    />
    <div class="flex-1">
      <KunTextarea v-model="content" :rows="3" />
      <div class="mt-3 flex justify-end">
        <KunButton
          :loading="isPublishing"
          :disabled="!content.trim() || isPublishing"
          @click="handlePublish"
        >
          {{ replyToPostId ? '发布回复' : '发表评论' }}
        </KunButton>
      </div>
    </div>
  </div>
</template>
