export interface ToolsetComment {
  id: number
  toolset_id: number
  created: Date | string
  content: string
  edited: Date | string | null
  parent_id: number | null
  user_id: number
  reply: ToolsetComment[]
  reply_count: number
  user: KunUser
  target_user?: KunUser | null
}
