export interface GalgameComment {
  id: number
  galgame_id: number
  // BE emits ISO timestamps (`Created string`, `Edited *string`); the
  // old `number` typings were a Nitro-era holdover. `formatTimeDifference`
  // and date helpers all accept ISO strings.
  created: string
  // Author-driven edit timestamp; null = never edited. Drives the
  // "已编辑" badge on the rendered comment.
  edited: string | null
  // Raw markdown source — round-trips through the edit-mode textarea.
  content: string
  // Server-rendered HTML via the project's goldmark pipeline (same as
  // topic / galgame intro / toolset / doc). Drop into <KunContent>;
  // DOMPurify there is the final XSS guard.
  content_html: string
  like_count: number
  is_liked: boolean

  user: KunUser
  target_user: KunUser | null

  // Nesting fields — populated by the threaded /comment/all endpoint.
  //
  // parentCommentId / rootCommentId are NULL on top-level comments;
  // a reply has parentCommentId set to its direct ancestor and
  // rootCommentId set to the thread's root id. Posting a reply sends
  // parentCommentId; the server derives rootCommentId.
  parent_comment_id: number | null
  root_comment_id: number | null
  // Direct + transitive descendant count. Drives the "查看更多 (N)"
  // load-more button at the depth cap.
  reply_count: number
  // Children pre-built by the server. Roots receive the full subtree;
  // the frontend caps visible recursion at depth 3 and routes deeper
  // exploration to ThreadDrawer.
  replies: GalgameComment[]
}
