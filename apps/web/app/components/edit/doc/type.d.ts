export type DocEditorMode = 'create' | 'rewrite'

export interface DocEditorForm {
  article_id: number | null
  title: string
  slug: string
  description: string
  banner: string
  banner_image_hash: string
  status: number
  is_pin: boolean
  content_markdown: string
  category_id: number
  tag_ids: number[]
}
