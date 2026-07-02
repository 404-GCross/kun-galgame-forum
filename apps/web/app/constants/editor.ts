import type { KunToolbarItem } from '@kungal/editor-vue'

// Shared <KunEditorToolbar> button order for EVERY editor in the forum (the
// topic-edit page + the reply/comment/toolset shim), so the toolbar layout is
// consistent site-wide — the reason @kungal/editor-nuxt added the `:items` prop
// (0.21.0). Media tools (image upload + emoji/sticker) are grouped right after
// 文本大小 (heading). `'|'` = divider; image/picker auto-drop without their
// adapter/feature and the surrounding dividers collapse.
export const KUN_EDITOR_TOOLBAR_ITEMS: KunToolbarItem[] = [
  'heading',
  '|',
  'image',
  'picker',
  '|',
  'bold',
  'italic',
  'strike',
  'code',
  '|',
  'bulletList',
  'orderedList',
  'quote',
  'codeBlock',
  'hr',
  '|',
  'spoiler'
]
