<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    target: CommunityCommentTarget
    replyToPostId?: number | null
    targetUserId?: number
    isReply?: boolean
  }>(),
  { replyToPostId: null, targetUserId: undefined, isReply: false }
)

const emits = defineEmits<{
  close: []
  submitted: [post: GalgameCommunityComment]
}>()

const surface = communityCommentSurface(props.target)

const { id } = usePersistUserStore()

const content = ref('')
const isPublishing = ref(false)

const handlePublish = async () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  const trimmed = content.value.trim()
  if (!trimmed) {
    useMessage(10540, 'warn')
    return
  }
  if (trimmed.length > surface.maxLength) {
    useMessage(`评论最大长度为 ${surface.maxLength} 个字符`, 'warn')
    return
  }

  isPublishing.value = true
  const result = await kunFetch<GalgameCommunityComment>(surface.listUrl, {
    method: 'POST',
    query: surface.addressQuery,
    body: surface.createBody(trimmed, props.replyToPostId, props.targetUserId)
  })
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
  <div class="space-y-3">
    <KunMilkdownDualEditorProvider
      :value-markdown="content"
      :placeholder="surface.composerPlaceholder"
      @set-markdown="(val) => (content = val)"
    />

    <div class="flex items-center justify-between gap-2">
      <slot />

      <KunButton class="ml-auto" :loading="isPublishing" @click="handlePublish">
        {{ isReply ? '发布回复' : '发布评论' }}
      </KunButton>
    </div>
  </div>
</template>
