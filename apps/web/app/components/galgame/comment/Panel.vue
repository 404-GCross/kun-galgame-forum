<script setup lang="ts">
// Compose-box for a new comment.
//
// Two modes:
//   - Root mode: no parentCommentId. Container renders this above the thread.
//   - Reply mode: parentCommentId set. Comment.vue renders this inline beneath
//     the comment being replied to (parent_comment_id threads it).
//
// Notifying a specific user is done by @-mentioning them in the editor (the
// legacy "评论给 xxx" target was retired) — the backend parses the mention.
//
// The editor is the project-wide KunMilkdownDualEditorProvider — same
// component used for topic / galgame intro / toolset / doc, so the
// authoring experience (toolbar, code-block tab, character counter,
// upload pipeline) is identical across the site. The wire payload is
// raw Markdown; the backend's goldmark renderer turns it into
// contentHtml on response.
const props = defineProps<{
  parentCommentId?: number | null
}>()

const emits = defineEmits<{
  close: []
  // Optimistic update: emit the server-returned new comment so the
  // parent (Container for roots, Comment for replies) can splice it
  // into the tree in place without re-fetching the entire page.
  submitted: [comment: GalgameComment]
}>()

const route = useRoute()

const content = ref('')
const galgameId = computed(() =>
  parseInt((route.params as { gid: string }).gid)
)
const isPublishing = ref(false)

const handlePublishComment = async () => {
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
  const result = await kunFetch<GalgameComment>(
    `/galgame/${galgameId.value}/comment`,
    {
      method: 'POST',
      body: {
        galgame_id: galgameId.value,
        parent_comment_id: props.parentCommentId ?? null,
        content: content.value
      }
    }
  )
  isPublishing.value = false

  if (result) {
    content.value = ''
    useMessage(10542, 'success')
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

      <KunButton
        class="ml-auto"
        :loading="isPublishing"
        @click="handlePublishComment"
      >
        {{ parentCommentId ? '发布回复' : '发布评论' }}
      </KunButton>
    </div>
  </div>
</template>
