<script setup lang="ts">
const props = defineProps<{
  reply: TopicReply
}>()

const { setRewriteData } = useTempReplyStore()
const { isEdit } = storeToRefs(useTempReplyStore())
const { id } = usePersistUserStore()

// Authors rewrite their own; staff holding reply.edit_any may rewrite anyone's.
// Mirrors 编辑任意话题 one level down — a moderator could already rewrite a whole
// topic but had to delete a reply outright to deal with one bad line in it.
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
  <KunTooltip v-if="isShowRewrite" text="重新编辑">
    <KunReaction
      :toggle="false"
      icon="lucide:pencil"
      label="重新编辑"
      @click="handleClickRewrite"
    />
  </KunTooltip>
</template>
