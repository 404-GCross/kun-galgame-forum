<script setup lang="ts">
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'
// The topic writing surface — a Tencent-Docs-style layout: a sticky formatting
// toolbar pinned at the top, the title below it, then the article body. All the
// publish metadata (category / section / NSFW / covers / read-rules) moved into
// <EditTopicPublishModal>, opened by the 发布话题 button.
//
// The editor's toolbar + view-switch are <KunEditor> scoped slots TELEPORTED
// into the sticky bar: Teleport moves only the DOM, so the slot `api` closures
// still drive the live editor. The toolbar is hidden in Markdown-source mode
// (its commands act on the WYSIWYG doc), tracked via `editorMode`.
const tempStore = useTempEditStore()
const { content } = useTopicEditorStore()
const adapters = useKunEditorAdapters()

const isPublishOpen = ref(false)
const openPublish = () => {
  isPublishOpen.value = true
}

const editorMode = ref<'wysiwyg' | 'source'>('wysiwyg')
const onSetMode = (
  mode: 'wysiwyg' | 'source',
  apply: (mode: 'wysiwyg' | 'source') => void
) => {
  editorMode.value = mode
  apply(mode)
}

// Ctrl+Enter opens the publish modal (category / section are chosen there now).
const onKeydown = (event: KeyboardEvent) => {
  if (event.ctrlKey && event.key === 'Enter') {
    event.preventDefault()
    openPublish()
  }
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))

onBeforeRouteLeave(async () => {
  // Vue Router 4: return false to stay, anything else to proceed.
  if (tempStore.isTopicRewriting) {
    const res = await useComponentMessageStore().alert(
      '确认离开界面吗？您的更改将不会保存。'
    )
    if (!res) {
      return false
    }
    tempStore.resetRewriteTopicData()
  }
})
</script>

<template>
  <div>
    <!-- Single REAL root box for the page transition — keep this comment INSIDE
         the root (a leading comment would be a second root node). -->
    <div
      class="et-topbar border-default-200 bg-background/90 sticky top-[72px] z-20 mb-4 flex items-center gap-3 rounded-lg border px-3 py-2 backdrop-blur"
    >
      <div id="et-viewswitch" class="shrink-0" />
      <div class="bg-default-200 h-5 w-px shrink-0" />
      <div id="et-toolbar" class="min-w-0 flex-1 overflow-x-auto" />
      <KunButton color="primary" class="shrink-0" @click="openPublish">
        <KunIcon name="lucide:send" class="mr-1 h-4 w-4" />
        发布话题
      </KunButton>
    </div>

    <div class="mx-auto max-w-3xl space-y-3">
      <EditTopicTitle />

      <ClientOnly>
        <KunEditor
          v-model="content"
          :adapters="adapters"
          class="kun-doc-editor"
        >
          <template #view-switch="api">
            <Teleport to="#et-viewswitch" defer>
              <KunEditorViewSwitch
                :mode="api.mode"
                :labels="api.labels"
                :set-mode="(mode) => onSetMode(mode, api.setMode)"
              />
            </Teleport>
          </template>
          <template #toolbar="api">
            <Teleport to="#et-toolbar" defer>
              <KunEditorToolbar v-if="editorMode === 'wysiwyg'" v-bind="api" />
            </Teleport>
          </template>
        </KunEditor>
      </ClientOnly>
    </div>

    <EditTopicPublishModal v-model="isPublishOpen" />
  </div>
</template>
