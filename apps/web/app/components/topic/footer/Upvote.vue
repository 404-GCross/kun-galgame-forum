<script setup lang="ts">
const props = defineProps<{
  topicId?: number
  targetUserId: number
  upvoteCount: number
  isUpvoted: boolean
  menu?: boolean
}>()

const { id, moemoepoint } = usePersistUserStore()
const isUpvoted = ref(props.isUpvoted)
const upvoteCount = ref(props.upvoteCount)

const { open } = useUpvoteModal()

const handleClickUpvote = async () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  if (id === props.targetUserId) {
    useMessage(10241, 'warn')
    return
  }
  if (moemoepoint < 10) {
    useMessage(10242, 'warn')
    return
  }
  if (!props.topicId) {
    return
  }
  const pushed = await open({
    topicId: props.topicId,
    targetUserId: props.targetUserId
  })
  if (pushed) {
    upvoteCount.value++
    isUpvoted.value = true
  }
}
</script>

<template>
  <KunButton
    v-if="menu"
    :variant="isUpvoted ? 'flat' : 'light'"
    :color="isUpvoted ? 'secondary' : 'default'"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    @click="handleClickUpvote"
  >
    <KunIcon class-name="text-lg" name="lucide:sparkles" />
    推话题
    <span v-if="upvoteCount" class="text-default-500 ml-auto">
      {{ upvoteCount }}
    </span>
  </KunButton>

  <KunTooltip v-else text="推话题">
    <KunReaction
      :toggle="false"
      :count="upvoteCount"
      label="推话题"
      @click="handleClickUpvote"
    >
      <template #icon>
        <KunIcon name="lucide:sparkles" class="text-warning" />
      </template>
    </KunReaction>
  </KunTooltip>
</template>
