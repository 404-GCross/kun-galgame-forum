<script setup lang="ts">
const props = defineProps<{
  comment: GalgameCommunityComment
}>()

const { id } = usePersistUserStore()
const isLiked = ref(props.comment.is_liked)
const likesCount = ref(props.comment.like_count)
const pending = ref(false)

const revert = (next: boolean) => {
  isLiked.value = !next
  likesCount.value += next ? -1 : 1
}

const onChange = async (next: boolean) => {
  if (!id) {
    useAuthModal().open()
    revert(next)
    return
  }
  if (id === props.comment.user.id) {
    useMessage(10533, 'warn')
    revert(next)
    return
  }

  pending.value = true
  const result = await kunFetch<{ liked: boolean; like_count: number }>(
    `/galgame/comments/${props.comment.id}/like`,
    { method: 'PUT' }
  )
  pending.value = false

  if (!result) {
    revert(next)
    return
  }
  isLiked.value = result.liked
  likesCount.value = result.like_count
}
</script>

<template>
  <KunTooltip text="点赞">
    <KunReaction
      v-model="isLiked"
      v-model:count="likesCount"
      :disabled="pending"
      size="sm"
      icon="lucide:thumbs-up"
      color="primary"
      label="点赞"
      @change="onChange"
    />
  </KunTooltip>
</template>
