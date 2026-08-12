<script setup lang="ts">
const props = defineProps<{
  reply: TopicReply
}>()

const { setRewriteData } = useTempReplyStore()
const { isEdit } = storeToRefs(useTempReplyStore())
const { id } = usePersistUserStore()

const canEditAnyReply = useCan('reply.edit_any')
const isShowRewrite = computed(
  () => id === props.reply.user.id || canEditAnyReply.value
)

const handleClickRewrite = () => {
  setRewriteData(props.reply)
  isEdit.value = true
}
</script>

<template>
  <KunButton
    v-if="isShowRewrite"
    variant="light"
    color="default"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    @click="handleClickRewrite"
  >
    <KunIcon class-name="text-lg" name="lucide:pencil" />
    重新编辑
  </KunButton>
</template>
