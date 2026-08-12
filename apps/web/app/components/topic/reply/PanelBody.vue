<script setup lang="ts">
const { isReplyRewriting, replyRewrite } = storeToRefs(useTempReplyStore())
const { replyDraft } = storeToRefs(usePersistKUNGalgameReplyStore())

const currentData = computed(() =>
  isReplyRewriting.value ? replyRewrite.value : replyDraft.value
)
</script>

<template>
  <div v-if="currentData" class="space-y-2">
    <TopicReplyBodyEditor
      :key="isReplyRewriting ? `edit-${replyRewrite?.id}` : 'draft'"
      v-model="currentData.mainContent"
    />

    <TopicReplyPanelBtn class="mt-3" />
  </div>
</template>
