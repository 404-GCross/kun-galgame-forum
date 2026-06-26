import type { SerializeObject } from 'nitropack'

// Flat-tier mutation helpers for the galgame comment tree.
//
// The visual model is intentionally TWO TIERS only: root + a flat list
// of replies. The DB still tracks the full parent_comment_id chain
// (so true "回复给某条" semantics survive via targetUser), but the
// frontend always renders root.replies as a single flat group with no
// further indent. These helpers therefore treat root.replies as a
// flat array — never walk inside a reply looking for sub-replies.
//
// Both Container.vue (inline list, ≤3 replies per root) and
// ThreadDrawer.vue (full thread, unbounded) call into the same logic.
// All returns are immutable: the input root is not mutated, and a new
// root object is returned so Vue can detect the change even when
// stored behind a shallowRef.

type Node = SerializeObject<GalgameComment>

// Append `newReply` at the BOTTOM of root.replies and bump replyCount.
// The inline + drawer views render replies oldest-first (先发布在上), so a
// just-posted reply — the newest — lands at the end to stay visible at the
// bottom of the list. Always returns a fresh root object.
export const appendReplyToRoot = (root: Node, newReply: Node): Node => {
  return {
    ...root,
    replies: [...(root.replies ?? []), newReply],
    replyCount: (root.replyCount ?? 0) + 1
  }
}

// Remove the reply matching `removedId` and (transitively) any of its
// descendants that happen to be loaded in the flat list. The DB
// cascades the actual rows via parent_comment_id FK; this mirrors that
// on the client so the visible list doesn't dangle.
//
// `removedSize` is the subtree size reported by the caller (the
// deleted node + all its DB descendants — most of which may not even
// be loaded). We decrement replyCount by that exact amount because
// the count is authoritative for the WHOLE thread, not just what's
// visible.
// Patch a single comment node in the tree (root or one of its
// flat-attached replies). Returns a fresh root with the matching
// node replaced; the rest of the tree is untouched. Used for the
// edit (PUT) path — the server response carries the freshly updated
// node and we splice it in by id.
export const replaceCommentInRoot = (
  root: Node,
  updated: Node
): { node: Node; replaced: boolean } => {
  // Did the root itself get edited?
  if (root.id === updated.id) {
    return {
      // Keep the original replies + replyCount on the new root — the
      // edit only touches the content+edited+target fields.
      node: {
        ...updated,
        replies: root.replies,
        replyCount: root.replyCount
      },
      replaced: true
    }
  }
  const replies = root.replies ?? []
  const idx = replies.findIndex((r) => r.id === updated.id)
  if (idx < 0) {
    return { node: root, replaced: false }
  }
  const nextReplies = [...replies]
  // Replies are flat in our view model and don't carry their own
  // replies/replyCount, so direct replacement is fine.
  nextReplies[idx] = updated
  return {
    node: { ...root, replies: nextReplies },
    replaced: true
  }
}

export const removeReplyFromRoot = (
  root: Node,
  removedId: number,
  removedSize: number
): { node: Node; pruned: boolean } => {
  const replies = root.replies ?? []
  if (!replies.some((r) => r.id === removedId)) {
    return { node: root, pruned: false }
  }

  // Collect removedId + any loaded descendants. parentCommentId chains
  // can be multi-hop within the loaded set (a depth-3 reply pointing
  // at a depth-2 reply), so iterate until the frontier stops growing.
  const toRemove = new Set<number>([removedId])
  let added = true
  while (added) {
    added = false
    for (const r of replies) {
      if (
        r.parentCommentId != null &&
        toRemove.has(r.parentCommentId) &&
        !toRemove.has(r.id)
      ) {
        toRemove.add(r.id)
        added = true
      }
    }
  }

  return {
    node: {
      ...root,
      replies: replies.filter((r) => !toRemove.has(r.id)),
      replyCount: Math.max(0, (root.replyCount ?? 0) - removedSize)
    },
    pruned: true
  }
}
