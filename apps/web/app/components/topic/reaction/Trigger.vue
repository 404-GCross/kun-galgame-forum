<script setup lang="ts">
// The reaction trigger: an icon-only KunReaction that opens the picker popover.
// State is injected (shared with the chips), so reacting here updates the bar.
const { toggle, mineKeys, list, topicId, replyId } = inject(reactionsKey)!

const picker = ref<{ close: () => void } | null>(null)
const historyOpen = ref(false)

// Which subject the history feed hangs off — absent where there is none.
const subject = topicId ? 'topic' : replyId ? 'reply' : undefined

// Total reactions on the subject (sum of per-reaction counts) — the "xxx 人" count.
const total = computed(() => list.value.reduce((sum, r) => sum + r.count, 0))

const onSelect = (key: string) => {
  picker.value?.close()
  toggle(key)
}

const onViewHistory = () => {
  picker.value?.close()
  historyOpen.value = true
}
</script>

<template>
  <KunPopover ref="picker" position="top-start" opaque>
    <!-- KunPopover wraps #trigger in an inline-block; a flex span here removes
         the inline line-box so the icon sits level with the sibling pills/chips
         (not nudged up by the baseline descender). -->
    <template #trigger>
      <span class="flex">
        <KunReaction :toggle="false" icon="lucide:smile-plus" label="表态" />
      </span>
    </template>

    <TopicReactionPicker
      :mine-keys="mineKeys"
      :subject="subject"
      :total="total"
      @select="onSelect"
      @view-history="onViewHistory"
    />
  </KunPopover>

  <TopicReactionHistoryModal
    v-if="subject"
    v-model="historyOpen"
    :topic-id="topicId"
    :reply-id="replyId"
  />
</template>
