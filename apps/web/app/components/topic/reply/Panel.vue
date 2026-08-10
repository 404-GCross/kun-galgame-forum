<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'

const tempReplyStore = useTempReplyStore()
const { isEdit, isReplyRewriting } = storeToRefs(tempReplyStore)

const isMobileQuery = useMediaQuery('(max-width: 767px)')
const mounted = ref(false)
onMounted(() => (mounted.value = true))
const isMobile = computed(() => mounted.value && isMobileQuery.value)

const handleDrawerClose = () => {
  if (isReplyRewriting.value) {
    tempReplyStore.resetRewriteReplyData()
  }
}
</script>

<template>
  <KunDrawer
    v-if="isMobile"
    v-model="isEdit"
    placement="bottom"
    size="lg"
    title="发表回复"
    @close="handleDrawerClose"
  >
    <TopicReplyPanelBody />
  </KunDrawer>

  <Teleport v-else to="body" :disabled="!isEdit">
    <Transition
      enter-active-class="animate-fadeInUp"
      leave-active-class="animate-fadeOutDown"
    >
      <div
        class="fixed bottom-0 z-100 flex max-h-[80%] w-full flex-col items-center"
        v-if="isEdit"
      >
        <div
          class="kun-reply-panel bg-content1 border-default/20 scrollbar-hide w-full max-w-4xl space-y-2 overflow-scroll rounded-t-lg border p-3"
        >
          <TopicReplyPanelBody />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
