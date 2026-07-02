<script setup lang="ts">
import type { SerializeObject } from 'nitropack'

const props = defineProps<{
  comment: SerializeObject<GalgameComment>
}>()

const { id } = usePersistUserStore()
const isLiked = ref(props.comment.is_liked)
const likesCount = ref(props.comment.like_count)
const pending = ref(false)

// Galgame comment likes are one-way (no un-like): KunReaction is disabled once
// liked, so @change only ever fires for the first like.
const revert = (next: boolean) => {
  isLiked.value = !next
  likesCount.value += next ? -1 : 1
}

const onChange = async (next: boolean) => {
  if (!next) return
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
  const result = await kunFetch(
    `/galgame/${props.comment.galgame_id}/comment/like`,
    { method: 'PUT', body: { comment_id: props.comment.id } }
  )
  pending.value = false
  if (!result) {
    revert(next)
    return
  }
  useMessage(10530, 'success')
}
</script>

<template>
  <KunTooltip text="点赞">
    <KunReaction
      v-model="isLiked"
      v-model:count="likesCount"
      :disabled="isLiked || pending"
      size="sm"
      icon="lucide:thumbs-up"
      color="primary"
      label="点赞"
      @change="onChange"
    />
  </KunTooltip>
</template>
