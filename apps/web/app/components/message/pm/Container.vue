<script setup lang="ts">
const props = defineProps<{
  userId: number
}>()

const historyScroll = useTemplateRef<{
  getViewport: () => HTMLElement | null
}>('historyScroll')
const getHistoryViewport = () => historyScroll.value?.getViewport() ?? null
const messageInput = ref('')
const messages = ref<ChatMessage[]>([])
const isLoadHistoryMessageComplete = ref(false)
const isSending = ref(false)
const isUploadingImage = ref(false)
const pendingImages = ref<{ name: string; url: string }[]>([])
const messageTextarea = useTemplateRef<{
  insertAtCaret: (text: string) => void
}>('messageTextarea')
const fileInput = ref<HTMLInputElement | null>(null)
const isShowLoader = computed(() => {
  if (isLoadHistoryMessageComplete.value) {
    return false
  }
  if (messages.value.length < 30) {
    return false
  }
  return true
})
const currentUserId = usePersistUserStore().id
const userId = props.userId
const pageData = reactive({
  page: 1,
  limit: 30
})

const scrollToBottom = () => {
  const viewport = getHistoryViewport()
  if (viewport) {
    viewport.scrollTo({
      top: viewport.scrollHeight,
      behavior: 'smooth'
    })
  }
}

const getMessageHistory = async () => {
  const histories = await kunFetch<ChatMessage[]>('/message/chat/history', {
    method: 'GET',
    query: {
      receiver_id: userId,
      page: pageData.page,
      limit: pageData.limit
    }
  })
  return Array.isArray(histories) ? histories : ([] as ChatMessage[])
}

const postMessage = async (content: string): Promise<boolean> => {
  if (isSending.value) {
    return false
  }
  if (content.length > 1000) {
    useMessage(10402, 'warn')
    return false
  }

  isSending.value = true
  const result = await kunFetch('/message/chat/send', {
    method: 'POST',
    body: { receiver_id: userId, content }
  })
  isSending.value = false

  if (!result) {
    return false
  }
  pageData.page = 1
  messages.value = await getMessageHistory()
  nextTick(() => scrollToBottom())
  return true
}

const sendMessage = async () => {
  if (isUploadingImage.value) {
    useMessage('图片正在上传中, 请稍候', 'warn')
    return
  }

  const text = messageInput.value.trim()
  const imageMarkdown = pendingImages.value
    .map((img) => `![${img.name.replace(/[[\]()]/g, '')}](${img.url})`)
    .join(' ')
  const content = [text, imageMarkdown].filter(Boolean).join('\n')

  if (!content) {
    useMessage(10401, 'warn')
    return
  }

  if (await postMessage(content)) {
    messageInput.value = ''
    pendingImages.value = []
  }
}

const uploadImages = async (files: File[]) => {
  const images = files.filter((file) => file.type.startsWith('image/'))
  if (!images.length) {
    return
  }

  isUploadingImage.value = true
  try {
    for (const image of images) {
      const formData = new FormData()
      formData.append('image', image)
      const url = await kunFetch<string>('/image/message', {
        method: 'POST',
        body: formData,
        watch: false
      })
      if (url) {
        pendingImages.value.push({ name: image.name, url })
      }
    }
  } finally {
    isUploadingImage.value = false
  }
}

const removePendingImage = (index: number) => {
  pendingImages.value.splice(index, 1)
}

const handlePaste = (event: ClipboardEvent) => {
  const files = Array.from(event.clipboardData?.files ?? [])
  if (!files.some((file) => file.type.startsWith('image/'))) {
    return
  }
  event.preventDefault()
  uploadImages(files)
}

const handleDrop = (event: DragEvent) => {
  uploadImages(Array.from(event.dataTransfer?.files ?? []))
}

const handleEnter = (event: KeyboardEvent) => {
  if (event.isComposing || event.shiftKey) {
    return
  }
  event.preventDefault()
  sendMessage()
}

const openFilePicker = () => {
  fileInput.value?.click()
}

const onFileChange = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.length) {
    uploadImages(Array.from(input.files))
  }
  input.value = ''
}

const onEmoji = (emoji: string) => {
  messageTextarea.value?.insertAtCaret(emoji)
}

const onSticker = (url: string) => {
  pendingImages.value.push({ name: 'sticker', url })
}

const handleRecallContextMenu = async (payload: {
  event: MouseEvent
  message: ChatMessage
}) => {
  const target = payload.message
  if (target.sender.id !== currentUserId || target.is_recall) {
    return
  }

  const confirmed = await useComponentMessageStore().alert(
    '撤回这条消息?',
    '撤回后对方将看到 “XX 撤回了一条消息”, 内容不可恢复'
  )
  if (!confirmed) {
    return
  }

  const ok = await kunFetch<string>('/message/chat/recall', {
    method: 'POST',
    body: { message_id: target.id }
  })
  if (!ok) {
    return
  }

  const idx = messages.value.findIndex((m) => m.id === target.id)
  if (idx !== -1) {
    messages.value[idx] = { ...messages.value[idx]!, is_recall: true }
  }
  useMessage('撤回成功', 'success')
}

const handleLoadHistoryMessages = async () => {
  const viewport = getHistoryViewport()
  if (!viewport) {
    return
  }

  const previousScrollHeight = viewport.scrollHeight
  const previousScrollTop = viewport.scrollTop

  pageData.page += 1
  const histories = await getMessageHistory()

  if (histories.length > 0) {
    messages.value.unshift(...histories)

    nextTick(() => {
      const next = getHistoryViewport()
      if (next) {
        const newScrollHeight = next.scrollHeight
        next.scrollTo({
          top: previousScrollTop + (newScrollHeight - previousScrollHeight)
        })
      }
    })
  } else {
    isLoadHistoryMessageComplete.value = true
  }
}

onMounted(async () => {
  messages.value = await getMessageHistory()

  nextTick(() => {
    scrollToBottom()
  })
})
</script>

<template>
  <KunOverlayScroll ref="historyScroll" :defer="false" class="min-h-0 flex-1">
    <div class="space-y-3 py-3">
      <div class="flex justify-center">
        <KunButton
          v-if="isShowLoader"
          @click="handleLoadHistoryMessages"
          size="sm"
          variant="light"
        >
          加载更多
        </KunButton>
      </div>

      <MessagePmItem
        v-for="message in messages"
        :key="message.id"
        :message="message"
        :is-sent="message.sender.id === currentUserId"
        @context-menu="handleRecallContextMenu"
      />

      <div v-if="!messages.length" class="text-default-500 py-10 text-center">
        暂无消息，发送一条消息开始聊天吧
      </div>
    </div>
  </KunOverlayScroll>

  <div
    class="shrink-0 border-t px-3 py-3"
    @paste="handlePaste"
    @drop.prevent="handleDrop"
    @dragover.prevent
  >
    <div
      v-if="pendingImages.length || isUploadingImage"
      class="mb-2 flex flex-wrap gap-2"
    >
      <div
        v-for="(img, index) in pendingImages"
        :key="img.url"
        class="border-default-200 relative h-16 w-16 overflow-hidden rounded-lg border"
      >
        <img :src="img.url" :alt="img.name" class="h-full w-full object-cover" />
        <button
          type="button"
          @click="removePendingImage(index)"
          class="bg-background/70 text-default-600 hover:text-danger absolute top-0.5 right-0.5 flex h-5 w-5 items-center justify-center rounded-full text-xs leading-none"
          aria-label="移除图片"
        >
          ✕
        </button>
      </div>
      <div
        v-if="isUploadingImage"
        class="border-default-200 text-default-500 flex h-16 w-16 items-center justify-center rounded-lg border border-dashed text-xs"
      >
        上传中...
      </div>
    </div>

    <div class="flex flex-col gap-1.5 sm:flex-row sm:items-end sm:gap-1">
      <div class="flex gap-1">
        <KunPopover position="top-start" :auto-position="true">
          <template #trigger>
            <KunButton
              :is-icon-only="true"
              variant="light"
              size="lg"
              aria-label="表情和贴纸"
            >
              <KunIcon name="lucide:smile" />
            </KunButton>
          </template>
          <MessagePmEmojiStickerPicker @emoji="onEmoji" @sticker="onSticker" />
        </KunPopover>

        <KunButton
          :is-icon-only="true"
          variant="light"
          size="lg"
          @click="openFilePicker"
          aria-label="上传图片"
        >
          <KunIcon name="lucide:image" />
        </KunButton>
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          multiple
          class="hidden"
          @change="onFileChange"
        />
      </div>

      <div class="flex flex-1 items-end gap-1">
        <KunTextarea
          ref="messageTextarea"
          v-model="messageInput"
          placeholder="输入消息... (可粘贴或拖拽图片, Enter 发送, Shift+Enter 换行)"
          class="flex-1"
          :auto-grow="true"
          :rows="1"
          max-height="160px"
          @keydown.enter="handleEnter"
        />
        <KunButton @click="sendMessage" :loading="isSending" size="lg">
          发送
        </KunButton>
      </div>
    </div>
  </div>
</template>
