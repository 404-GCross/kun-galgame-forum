export interface ToolsetComment {
  id: number
  toolsetId: number
  created: Date | string
  content: string
  edited: Date | string | null
  parentId: number | null
  userId: number
  // Roots carry their full set of descendants flattened here (oldest-first, one
  // visual tier); replies leave it empty. replyCount = reply.length (all loaded;
  // "展开更多" reveals them inline, no lazy fetch).
  reply: ToolsetComment[]
  replyCount: number
  user: KunUser
  targetUser?: KunUser | null
}
