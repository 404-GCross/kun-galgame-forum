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
  <!-- A ⋯ menu row, like its siblings 置顶 / 最佳答案 / 删除: editing is an owner
       action, not something every reader needs a button for. -->
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
