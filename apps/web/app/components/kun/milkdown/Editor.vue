<script setup lang="ts">
// Milkdown core
import {
  Editor,
  rootCtx,
  defaultValueCtx,
  editorViewCtx
} from '@milkdown/kit/core'
import type { EditorView as PmEditorView } from '@milkdown/prose/view'
import { Milkdown, useEditor } from '@milkdown/vue'
import type { ReplyReference } from '~/store/types/topic/reply'
import { commonmark } from '@milkdown/kit/preset/commonmark'
import { gfm } from '@milkdown/kit/preset/gfm'
// Milkdown Plugins
import { history } from '@milkdown/kit/plugin/history'
import { listener, listenerCtx } from '@milkdown/kit/plugin/listener'
import { clipboard } from '@milkdown/kit/plugin/clipboard'
import { indent } from '@milkdown/kit/plugin/indent'
import { trailing } from '@milkdown/kit/plugin/trailing'
import { usePluginViewFactory } from '@prosemirror-adapter/vue'
import {
  upload,
  uploadConfig,
  type Uploader
} from '@milkdown/kit/plugin/upload'

// Custom plugins
import { activeTab } from './atom'
import { kunUploader, kunUploadWidgetFactory } from './plugins/upload/uploader'
import { tooltipFactory } from '@milkdown/kit/plugin/tooltip'
import Tooltip from './plugins/tooltip/Tooltip.vue'
import { slashFactory } from '@milkdown/kit/plugin/slash'
import MentionDropdown from './plugins/mention/MentionDropdown.vue'
import { replaceAll } from '@milkdown/kit/utils'
import {
  stopLinkCommand,
  linkCustomKeymap
} from './plugins/stop-link/stopLinkPlugin'
import { kunSpoilerPlugin } from './plugins/spoiler/spoilerPlugin'
import { kunMentionPlugin } from './plugins/mention/mentionPlugin'
import { kunQuotePlugin } from './plugins/quote/quotePlugin'

// Code Block
import { defaultKeymap, indentWithTab } from '@codemirror/commands'
import { keymap, EditorView } from '@codemirror/view'
import {
  codeBlockComponent,
  codeBlockConfig,
  type CodeBlockConfig
} from '@milkdown/kit/component/code-block'
import { basicSetup } from 'codemirror'
import {
  chevronDownIcon,
  clearIcon,
  copyIcon,
  editIcon,
  searchIcon,
  visibilityOffIcon
} from './plugins/code/icons'
import { languages } from '@codemirror/language-data'
import { kunCM } from './codemirror/theme'

// katex
import { blockKatexSchema } from './plugins/katex/blockKatex'
import { mathInlineSchema } from './plugins/katex/inlineKatex'
import { toggleLatexCommand } from './plugins/katex/command'
import {
  mathBlockInputRule,
  mathInlineInputRule
} from './plugins/katex/inputRule'
import { remarkMathBlockPlugin, remarkMathPlugin } from './plugins/katex/remark'
import katex from 'katex'
import type { KatexOptions } from 'katex'

const props = defineProps<{
  valueMarkdown: string
  language: Language
  // When true, all image-insertion paths are removed: the toolbar upload
  // button, paste/drop upload, AND the sticker inserter (stickers render as
  // images). Used by the galgame 简介 editor — intro images now live in the
  // wiki gallery, so the introduction itself must stay image-free.
  disableImage?: boolean
  // A 「引用」 signal (reply editor only): when set, the editor inserts an inline
  // `@author #floor` header into its live document and emits `quoteInserted` so
  // the parent can clear it. Other editors never pass this.
  pendingQuote?: ReplyReference | null
}>()

const emits = defineEmits<{
  saveMarkdown: [markdown: string]
  quoteInserted: []
}>()

const valueMarkdown = computed(() => props.valueMarkdown)

const tooltip = tooltipFactory('Text')
// @mention autocomplete: @ trigger → debounced /user/search → insert mention node.
const mentionSlash = slashFactory('kunMention')
const pluginViewFactory = usePluginViewFactory()
const container = ref<HTMLElement | null>(null)
const toolbar = ref<HTMLElement | null>(null)
// The markdown the editor last emitted; lets the valueMarkdown watch below tell
// an EXTERNAL change to the bound model (a parent replacing it) from the editor's
// own edits, so only the former triggers a replaceAll (the latter would reset the
// cursor every keystroke).
let lastEmitted = props.valueMarkdown

const renderLatex = (content: string, options?: KatexOptions) => {
  const html = katex.renderToString(content, {
    ...options,
    throwOnError: false,
    displayMode: true
  })
  return html
}

// Insert "@author #floor " at the cursor in the LIVE, focused editor — the same
// transaction the @-mention dropdown uses (which is caret-reliable, incl. IME).
//
// The TRAILING SPACE after the #quote atom is the whole fix: a paragraph that
// ENDS in a non-editable inline atom has no stable caret position after it, so
// the caret — and the next keystroke, especially an IME composition — snaps to
// BEFORE the atom (the bug: text landed back on the @ line). A real text node
// (the space) after the atom gives the caret somewhere to anchor. Deduped on
// reply id.
const insertReference = (view: PmEditorView, q: ReplyReference) => {
  const { state } = view
  const mentionType = state.schema.nodes.mention
  const quoteType = state.schema.nodes.quote
  if (!mentionType || !quoteType) {
    return
  }

  let already = false
  state.doc.descendants((node) => {
    if (node.type.name === 'quote' && node.attrs.replyId === q.replyId) {
      already = true
      return false
    }
    return undefined
  })
  if (already) {
    return
  }

  const fragment = [
    mentionType.create({ userId: q.userId, name: q.userName }),
    state.schema.text(' '),
    quoteType.create({ replyId: q.replyId, floor: q.floor }),
    state.schema.text(' ')
  ]
  const { from, to } = state.selection
  // replaceWith leaves the caret right after the inserted content (the trailing
  // space), where it anchors cleanly.
  const tr = state.tr.replaceWith(from, to, fragment)
  view.focus()
  view.dispatch(tr.scrollIntoView())
}

// Consume the pending 「引用」 signal against the live view, then tell the parent
// to clear it. Deferred to the next frame so the (freshly Teleported) editor is
// painted + focusable before we insert.
const consumePendingQuote = (view: PmEditorView) => {
  const q = props.pendingQuote
  if (!q) {
    return
  }
  requestAnimationFrame(() => {
    insertReference(view, q)
    emits('quoteInserted')
  })
}

const editorInfo = useEditor((root) => {
  const editor = Editor.make()
    .config((ctx) => {
      ctx.set(rootCtx, root)
      ctx.set(defaultValueCtx, valueMarkdown.value)

      const listener = ctx.get(listenerCtx)
      listener.markdownUpdated((ctx, markdown, prevMarkdown) => {
        if (markdown !== prevMarkdown) {
          // Record what the editor produced so the valueMarkdown watch below can
          // tell an external change (a parent replacing the bound model) from the
          // editor's own edits and avoid a replaceAll feedback loop.
          lastEmitted = markdown
          emits('saveMarkdown', markdown)
        }
      })
      // Fresh-open case: 「引用」 set pendingQuote BEFORE the panel (and this
      // editor) mounted, so insert the header once the view is ready.
      listener.mounted((ctx) => {
        consumePendingQuote(ctx.get(editorViewCtx))
      })

      // Only wire the uploader when images are allowed. uploadConfig.key is
      // registered by the `upload` plugin, which we omit below when
      // disableImage is set — touching the key without the plugin would throw.
      if (!props.disableImage) {
        ctx.update(uploadConfig.key, (prev) => ({
          ...prev,
          // kunUploader is typed against @milkdown/plugin-upload's Uploader, whose
          // Ctx resolves to a different (duplicated) @milkdown/ctx copy than kit's
          // re-export. The two Ctx types are runtime-identical but nominally
          // distinct (private brand), so go through `unknown` to unify them.
          uploader: kunUploader as unknown as Uploader,
          uploadWidgetFactory: kunUploadWidgetFactory
        }))
      }

      ctx.set(tooltip.key, {
        view: pluginViewFactory({
          component: Tooltip
        })
      })

      ctx.set(mentionSlash.key, {
        view: pluginViewFactory({
          component: MentionDropdown
        })
      })

      const extensions = [
        kunCM(),
        EditorView.lineWrapping,
        keymap.of(defaultKeymap.concat(indentWithTab)),
        basicSetup
      ]
      // if (theme) {
      //   extensions.push(theme)
      // }

      ctx.update(codeBlockConfig.key, (defaultConfig) => ({
        extensions,
        languages,
        expandIcon: chevronDownIcon,
        searchIcon: searchIcon,
        clearSearchIcon: clearIcon,
        searchPlaceholder: '搜索咒文',
        copyText: '复制咒文',
        copyIcon: copyIcon,
        onCopy: () => {},
        noResultText: '无结果',
        renderLanguage: defaultConfig.renderLanguage,
        previewLoading: '加载中...',
        renderPreview: defaultConfig.renderPreview,
        previewToggleButton: (previewOnlyMode) => {
          const icon = previewOnlyMode ? editIcon : visibilityOffIcon
          const text = previewOnlyMode ? '编辑' : '隐藏'
          return [icon, text].map((v) => v.trim()).join(' ')
        },
        previewLabel: defaultConfig.previewLabel
        // previewLoading: config.previewLoading || defaultConfig.previewLoading,
        // previewOnlyByDefault:
        //   config.previewOnlyByDefault ?? defaultConfig.previewOnlyByDefault
      }))

      const katexOptions: KatexOptions = {}

      ctx.update(codeBlockConfig.key, (prev) => ({
        ...prev,
        renderPreview: (language, content, applyPreview) => {
          if (language.toLowerCase() === 'latex' && content.length > 0) {
            return renderLatex(content, katexOptions)
          }
          const renderPreview = prev.renderPreview
          return renderPreview(language, content, applyPreview)
        }
      }))
    })
    .use(history)
    .use(commonmark)
    .use(gfm)
    .use(listener)
    .use(clipboard)
    .use(indent)
    .use(trailing)
    .use(tooltip)
    .use(mentionSlash)
    .use(codeBlockComponent)
    .use([kunSpoilerPlugin, stopLinkCommand, linkCustomKeymap].flat())
    .use(kunMentionPlugin)
    .use(kunQuotePlugin)
    .use(remarkMathPlugin)
    .use(remarkMathBlockPlugin)
    .use(mathInlineSchema)
    .use(mathInlineInputRule)
    .use(mathBlockInputRule)
    .use(blockKatexSchema)
    .use(toggleLatexCommand)

  // Image upload (toolbar + paste/drop) is opt-out: omit the `upload` plugin
  // entirely for image-free editors (galgame 简介). The toolbar button and the
  // sticker inserter are hidden separately via the menu's allow-image prop.
  if (!props.disableImage) {
    editor.use(upload)
  }

  return editor
})

const textCount = computed(() => markdownToText(props.valueMarkdown).length)

// Only touch the WYSIWYG view while it's actually mounted. `<Milkdown>` is
// v-if'd out in 'code' (markdown source) mode, so calling .action() on the
// torn-down editor throws ("editorView not found" / null parentNode) — that was
// the crash when editing in markdown mode. Switching back to preview remounts the
// editor and re-syncs from defaultValueCtx, so skipping here loses nothing.
const onPreviewEditor = (run: (editor: Editor) => void) => {
  if (activeTab.value !== 'preview') {
    return
  }
  const editor = editorInfo.get()
  if (!editor) {
    return
  }
  try {
    run(editor)
  } catch {
    // The view may be mid-(un)mount around a tab switch; the next mount re-syncs.
  }
}

watch(
  () => [props.language],
  () => {
    onPreviewEditor((editor) => editor.action(replaceAll(valueMarkdown.value)))
  }
)

// Re-sync the doc when valueMarkdown changes from OUTSIDE the editor (a parent
// replacing the bound model). Guarded by lastEmitted so the editor's own edits
// (which round-trip back through the model) don't trigger a replaceAll and reset
// the cursor on every keystroke.
watch(
  () => props.valueMarkdown,
  (val) => {
    if (val === lastEmitted) {
      return
    }
    lastEmitted = val
    onPreviewEditor((editor) => editor.action(replaceAll(val)))
  }
)

// Live-append case: 「引用」 clicked while the panel is already open (editor
// mounted). Insert the header into the live view. (Fresh-open is handled by
// listener.mounted above; this watch is non-immediate so it won't double-fire
// for a value that was already set at mount.)
watch(
  () => props.pendingQuote,
  (q) => {
    if (!q) {
      return
    }
    onPreviewEditor((editor) =>
      editor.action((ctx) => consumePendingQuote(ctx.get(editorViewCtx)))
    )
  }
)
</script>

<template>
  <div ref="container" class="space-y-3">
    <KunMilkdownPluginsMenu
      ref="toolbar"
      :editor-info="editorInfo"
      :allow-image="!disableImage"
    />

    <template v-if="activeTab === 'preview'">
      <!-- KunOverlayScroll wraps <Milkdown /> as an ANCESTOR (.os-host >
           .os-viewport > .os-content > .milkdown), so ProseMirror's own DOM is
           left untouched. The 500px cap + scrolling moved here off .milkdown. -->
      <KunOverlayScroll class="max-h-[500px]">
        <Milkdown />
      </KunOverlayScroll>

      <div class="flex items-center justify-between text-sm">
        <slot name="footer" />

        <div class="flex shrink-0 items-center gap-2">
          <KunChip color="success">
            <KunIcon
              name="simple-icons:markdown"
              class="text-success-700 dark:text-success"
            />
            Markdown 支持
          </KunChip>
          <span>
            {{ `${textCount} 字` }}
          </span>
        </div>
      </div>

      <div class="text-default-500 text-sm">
        特殊语法: 您可以使用 ||隐藏文本|| 来隐藏图片或者文字 (目前依然禁止 R18
        图片内容)
      </div>
    </template>
  </div>
</template>
