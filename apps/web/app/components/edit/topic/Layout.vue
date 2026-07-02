<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'
import { KUN_EDITOR_TOOLBAR_ITEMS } from '~/constants/editor'
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

type EditorViewMode = 'wysiwyg' | 'source' | 'split'
const editorMode = ref<EditorViewMode>('wysiwyg')
const onSetMode = (
  mode: EditorViewMode,
  apply: (mode: EditorViewMode) => void
) => {
  editorMode.value = mode
  apply(mode)
}

// Split view is desktop-only — it needs width for two panes and stacks awkwardly
// on a phone, so drop it from the offered view modes below 768px.
const isMobile = useMediaQuery('(max-width: 767px)')
const editorViews = computed<EditorViewMode[]>(() =>
  isMobile.value ? ['wysiwyg', 'source'] : ['wysiwyg', 'source', 'split']
)
// Keep the local mirror in sync with <KunEditor>'s own fallback: when split stops
// being offered (a resize to mobile while in split), it falls back to the first
// offered view (wysiwyg).
watch(isMobile, (mobile) => {
  if (mobile && editorMode.value === 'split') {
    editorMode.value = 'wysiwyg'
  }
})

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
         the root (a leading comment would be a second root node).

         Sticky formatting bar. It wraps responsively: on desktop it's one row
         (view-switch · toolbar · 发布); on mobile the toolbar drops to its own
         full-width, horizontally-scrollable row so it never wraps into a tall
         block (view-switch + 发布 stay on the top row). -->
    <div
      class="et-topbar border-default-200 bg-background/90 sticky top-[72px] z-20 mb-4 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg border px-3 py-2 backdrop-blur"
    >
      <div id="et-viewswitch" class="shrink-0" />
      <span class="bg-default-200 hidden h-5 w-px shrink-0 sm:block" />
      <KunScrollShadow
        axis="horizontal"
        shadow-size="1.5rem"
        shadow-color="var(--color-default-100)"
        draggable
        class="bg-default-100 order-last w-full rounded-lg px-1 py-1 sm:order-none sm:w-auto sm:min-w-0 sm:flex-1 sm:rounded-none sm:bg-transparent sm:p-0"
      >
        <div id="et-toolbar" class="w-max" />
      </KunScrollShadow>
      <KunButton
        color="primary"
        class="order-1 ml-auto shrink-0 sm:order-none sm:ml-0"
        @click="openPublish"
      >
        <KunIcon name="lucide:send" class="mr-1 h-4 w-4" />
        发布话题
      </KunButton>
    </div>

    <!-- The document card: title + body share one shadowed surface, with a faint
         divider between them; neither draws its own border/shadow. -->
    <!-- The doc card is a centered reading column (max-w-3xl), but in split mode
         it widens to the full topbar/content width so the two panes get room. -->
    <div :class="cn('mx-auto', editorMode === 'split' ? 'max-w-none' : 'max-w-3xl')">
      <div class="bg-content1 shadow-kun-md px-6 sm:px-16">
        <div class="border-default-200/60 border-b pt-8 pb-4 sm:pt-14 sm:pb-5">
          <EditTopicTitle />
        </div>

        <div class="pt-5 pb-10 sm:pt-7 sm:pb-20">
          <ClientOnly>
            <KunEditor
              v-model="content"
              :adapters="adapters"
              :views="editorViews"
              placeholder="在此输入您的话题正文..."
              class="kun-doc-editor"
            >
              <template #view-switch="api">
                <Teleport to="#et-viewswitch" defer>
                  <KunEditorViewSwitch
                    v-bind="api"
                    :set-mode="(mode) => onSetMode(mode, api.setMode)"
                  />
                </Teleport>
              </template>
              <template #toolbar="api">
                <Teleport to="#et-toolbar" defer>
                  <KunEditorToolbar
                    v-if="editorMode === 'wysiwyg'"
                    v-bind="api"
                    :items="KUN_EDITOR_TOOLBAR_ITEMS"
                  />
                </Teleport>
              </template>
            </KunEditor>
          </ClientOnly>
        </div>
      </div>
    </div>

    <EditTopicPublishModal v-model="isPublishOpen" />
  </div>
</template>

<!-- Global (not scoped): the toolbar is TELEPORTED into #et-toolbar, so a scoped
     rule wouldn't reach it. <KunEditorToolbar>'s root is `flex flex-wrap`; force
     a single row here so #et-toolbar's overflow-x-auto scrolls it instead of it
     stacking into multiple tall rows (the mobile problem). -->
<style>
#et-toolbar > * {
  flex-wrap: nowrap;
}
</style>
