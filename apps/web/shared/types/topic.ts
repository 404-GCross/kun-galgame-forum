export interface TopicCard {
  id: number
  title: string
  view: number
  tag: string[]
  section: string[]
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
  created: Date | string
  upvoteTime: Date | string | null
}

export interface TopicAside {
  title: string
  tid: number
}

// Slim best-answer projection embedded directly in TopicDetail (BE
// populates from topic.best_answer_id during /topic/:tid). Used by the
// detail page to render JSON-LD `acceptedAnswer` schema during SSR
// without a second /reply fetch.
export interface TopicBestAnswerSummary {
  id: number
  floor: number
  user: KunUser
  contentMarkdown: string
  contentHtml: string
  created: Date | string
}

export interface TopicDetail {
  id: number
  title: string
  contentMarkdown: string
  contentHtml: string
  view: number
  status: number
  isNSFW: boolean
  category: string
  section: string[]
  tag: string[]
  // Optional 1..9 cover images as /image/<hash> tokens; used to prefill the
  // cover picker when editing, and (empty = none) for the feed card.
  coverImages: string[]
  // See TopicCard.coverImageMeta — keyed by the /image/<hash> cover token.
  coverImageMeta?: Record<string, KunImageMeta>
  user: KunUser & { moemoepoint: number }

  likeCount: number
  isLiked: boolean
  dislikeCount: number
  isDisliked: boolean
  favoriteCount: number
  isFavorited: boolean
  upvoteCount: number
  isUpvoted: boolean
  reactions: KunReaction[]

  replyCount: number
  isPollTopic: boolean

  statusUpdateTime: Date | string
  upvoteTime: Date | string | null
  edited: Date | string | null
  created: Date | string

  // Embedded by BE when topic.best_answer_id is set; omitted otherwise.
  // Drives JSON-LD `acceptedAnswer` for SEO on the topic detail page.
  bestAnswer?: TopicBestAnswerSummary
}
