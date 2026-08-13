<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'

const props = defineProps<{
  message: ChatMessage
  isSent: boolean
}>()

const emit = defineEmits<{
  (
    event: 'context-menu',
    payload: { event: MouseEvent; message: ChatMessage }
  ): void
}>()

const contentRef = ref<HTMLElement | null>(null)
const { isLightboxOpen, images, currentImageIndex } =
  useContentLightbox(contentRef)

const isMobile = useMediaQuery('(max-width: 640px)')
const canRecall = computed(() => props.isSent && !props.message.is_recall)
const recallText = computed(() => `${props.message.sender.name}撤回了一条消息`)
const recallCursorClass = computed(() => {
  if (!canRecall.value) {
    return ''
  }

  return isMobile.value ? 'cursor-pointer' : 'cursor-context-menu'
})

const handleContextMenu = (event: MouseEvent) => {
  if (!canRecall.value || isMobile.value) {
    return
  }
  event.stopPropagation()
  emit('context-menu', { event, message: props.message })
}

const handleClick = (event: MouseEvent) => {
  if (!canRecall.value || !isMobile.value) {
    return
  }
  event.stopPropagation()
  emit('context-menu', { event, message: props.message })
}
</script>

<template>
  <div
    class="flex w-full"
    :class="[
      message.is_recall
        ? 'items-center justify-center py-2'
        : 'items-end gap-2',
      message.is_recall ? '' : isSent ? 'flex-row-reverse' : 'flex-row'
    ]"
  >
    <template v-if="message.is_recall">
      <span
        class="bg-default-100 text-default-500 rounded-full px-3 py-1 text-xs sm:text-sm"
      >
        {{ recallText }}
      </span>
    </template>

    <template v-else>
      <KunAvatar
        :disable-floating="true"
        :user="message.sender"
        class="mb-auto"
      />

      <div
        class="relative max-w-[75%] rounded-lg border p-3 transition-colors"
        :class="[
          isSent
            ? 'bg-primary/20 border-primary/20'
            : 'bg-background border-default-300',
          recallCursorClass
        ]"
        @contextmenu.prevent="handleContextMenu"
        @click="handleClick"
      >
        <div class="flex items-end">
          <span
            class="text-sm font-medium"
            :class="isSent ? 'text-primary' : 'text-secondary'"
          >
            {{ message.sender.name }}
          </span>
        </div>

        <div class="mt-1 text-sm leading-relaxed">
          <div
            ref="contentRef"
            class="kun-message-content [&_a]:text-primary [&_code]:bg-default-200/70 break-words [&_a]:underline [&_code]:rounded [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[0.85em] [&_img]:my-1 [&_img]:max-h-60 [&_img]:max-w-full [&_img]:cursor-zoom-in [&_img]:rounded-lg [&_p]:m-0 [&_strong]:font-semibold"
            v-html="message.content_html"
          />
          <div class="text-default-500 mt-0.5 text-right text-xs">
            <KunTime :time="message.created" />
          </div>
        </div>
      </div>
    </template>

    <KunLightbox
      v-model:is-open="isLightboxOpen"
      :images="images"
      :initial-index="currentImageIndex"
    />
  </div>
</template>
