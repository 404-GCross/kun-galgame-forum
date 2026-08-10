export interface ChatMessageHistoryRequest {
  receiverId: string
  page: string
  limit: string
}

export interface ChatMessageAsideItem {
  chatroom_name: string
  content: string
  count: number
  unread_count: number
  route: string
  title: string
  avatar: string
  last_message_time: Date | string | null
}

export interface ChatMessage {
  id: number
  chatroom_name: string
  sender: KunUser
  read_by: KunUser[]
  receiver_id: number
  content: string
  content_html: string
  is_recall: boolean
  created: Date | string
  recall_time: Date | string | null
  edit_time: Date | string | null
}
