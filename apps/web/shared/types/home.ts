import type { GalgameCard } from './galgame.d.ts'

export interface HomeUserStatus {
  moemoepoints: number
  isCheckIn: boolean
  hasNewMessage: boolean
}

export interface HomeTopic {
  id: number
  title: string
  view: number

  section: string[]
  tag: string[]
  // Optional 1..9 feed-card cover images, as /image/<hash> content tokens
  // (usable directly as an <img src>). Empty array = no covers.
  coverImages: string[]
  // Per-cover-token metadata (dims + ThumbHash), keyed by the /image/<hash>
  // token in coverImages — for no-CLS aspect ratio + blur-up. Absent pre-backfill.
  coverImageMeta?: Record<string, KunImageMeta>
  user: KunUser
  status: number
  hasBestAnswer: boolean
  isPollTopic: boolean
  isNSFWTopic: boolean

  likeCount: number
  replyCount: number
  commentCount: number

  statusUpdateTime: Date | string
  upvoteTime: Date | string | null
}

export type HomeGalgame = GalgameCard
