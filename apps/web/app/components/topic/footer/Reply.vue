<script setup lang="ts">
const { id } = usePersistUserStore()
const tempReplyStore = useTempReplyStore()
const { isEdit } = storeToRefs(tempReplyStore)

const props = defineProps<{
  targetUserName: string
  targetUserId: number
  targetFloor: number
  targetReplyId?: number
}>()

const handleClickReply = () => {
  if (!id) {
    useAuthModal().open()
    return
  }

  if (props.targetFloor !== 0 && props.targetReplyId) {
    tempReplyStore.setPendingQuote({
      userId: props.targetUserId,
      userName: props.targetUserName,
      replyId: props.targetReplyId,
      floor: props.targetFloor
    })
  }

  isEdit.value = true
}
</script>

<template>
  <KunReaction :toggle="false" icon="lucide:reply" @click="handleClickReply">
    回复
  </KunReaction>
</template>
