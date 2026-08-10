export interface TopicComment {
  id: number
  reply_id: number
  topic_id: number
  parent_comment_id?: number | null
  user: KunUser
  target_user: KunUser
  content: string
  is_liked: boolean
  like_count: number
  created: Date | string
  edited?: Date | string | null
}
