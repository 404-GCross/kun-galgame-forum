export type MessageType =
  | 'upvoted'
  | 'liked'
  | 'favorite'
  | 'replied'
  | 'solution'
  | 'pin-reply'
  | 'commented'
  | 'expired'
  | 'requested'
  | 'merged'
  | 'declined'
  | 'mentioned'
  | 'admin'
  | 'quiz-answered'

type MessageStatus = 'read' | 'unread'

export interface NotificationPreference {
  muted_types: string[]
}

type MessageSortField = 'time'

export interface MessageRequestData {
  page: string
  limit: string
  type?: MessageType | ''
  sortField?: MessageSortField
  sortOrder: KunOrder
}

export interface Message {
  id: number
  sender: KunUser
  receiver_id: number
  link: string
  content: string
  status: MessageStatus
  type: MessageType
  created: Date | string
}

export interface MessageList {
  messages: Message[]
  total: number
}

export interface MessageSystemMessage {
  id: number
  is_read: boolean
  content: KunNullable<KunLanguage>
  admin: KunUser
  created: Date | string
}
