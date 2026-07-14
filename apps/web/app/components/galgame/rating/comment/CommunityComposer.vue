<script setup lang="ts">
// Compose-box for a community-backed rating comment (charter step 08). The rating
// area is FLAT: every comment is a root post, but the primitive stores an explicit
// target_user_id ("A → B"), so this composer ALWAYS carries a target — the rating
// author for a top-level comment, the replied comment's author for a reply. Plain
// KunTextarea authoring (parity with the legacy Panel — the resource areas were
// never markdown, so we keep them plain rather than adopting galgame's editor).
const props = defineProps<{
  ratingId: number
  // Required recipient (charter ruling 19 — the rating create rejects a missing
  // target_user_id). Top-level = the rating author; reply = the parent's author.
  targetUserId: number
}>()

const emits = defineEmits<{
  close: []
  // Optimistic update: the server-returned post so the parent can splice it into
  // the flat list without re-fetching. A held (TL0) post still comes back so its
  // author sees their own "审核中" comment.
  submitted: [post: GalgameCommunityComment]
}>()

const content = ref('')
const isPublishing = ref(false)

const handlePublish = async () => {
  const text = content.value.trim()
  if (!text) {
    useMessage('请输入评论内容', 'warn')
    return
  }
  if (text.length > 1314) {
    useMessage('内容长度不能超过 1314 字', 'warn')
    return
  }

  isPublishing.value = true
  const result = await kunFetch<GalgameCommunityComment>(
    `/galgame-rating/${props.ratingId}/comments`,
    {
      method: 'POST',
      body: { content: text, target_user_id: props.targetUserId }
    }
  )
  isPublishing.value = false

  if (result) {
    content.value = ''
    // A TL0 first post is held for review; the receipt says so, but we still
    // insert it locally (the author sees their own held post).
    useMessage(result.held ? '已提交，待审核' : '发布成功', 'success')
    emits('submitted', result)
    emits('close')
  }
}
</script>

<template>
  <div class="space-y-3">
    <KunTextarea v-model="content" name="rating-comment" :rows="5" />

    <div class="flex items-center justify-end">
      <KunButton class="ml-auto" :loading="isPublishing" @click="handlePublish">
        发布
      </KunButton>
    </div>
  </div>
</template>
