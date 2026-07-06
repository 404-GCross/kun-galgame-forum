<script setup lang="ts">
import type { SerializeObject } from 'nitropack'

const route = useRoute()
const gid = parseInt((route.params as { gid: string }).gid)

// See GalgameResource: for a wiki-catalogue game the forum hasn't ingested, hide
// the empty-state (the detail page's 未收录 notice covers it) but KEEP the comment
// input — commenting creates the local row, so it's part of the recording funnel.
const galgame = inject<GalgameDetail>('galgame')

const pageData = reactive({
  galgame_id: gid,
  page: 1,
  limit: 30,
  // Roots default oldest-first (先发布在上); the toggle can flip it.
  sort_order: 'asc'
})

// data.items holds ROOTS only; each root carries up to 3 of its most
// recent descendants flattened into a single .replies list (any DB
// depth, displayed as one visual tier). Everything older is reached
// lazily through ThreadDrawer.
//
// Mutations after the initial load are applied OPTIMISTICALLY by
// rewriting `data.value` to a fresh object — no `refresh()`, no
// loading flash.
const { data, status } = await useKunFetch<{
  items: GalgameComment[]
  total: number
}>(`/galgame/${gid}/comment/all`, {
  lazy: true,
  method: 'GET',
  query: pageData
})

type CommentNode = SerializeObject<GalgameComment>

// ──────────────────────────────────────────
// Optimistic update handlers
// ──────────────────────────────────────────

const handleNewComment = (newComment: GalgameComment) => {
  if (!data.value) return
  const reply = newComment as CommentNode

  // Root: insert at the visible end of the current sort.
  if (reply.parent_comment_id == null) {
    const nextItems =
      pageData.sort_order === 'desc'
        ? [reply, ...data.value.items]
        : [...data.value.items, reply]
    data.value = { items: nextItems, total: (data.value.total ?? 0) + 1 }
    return
  }

  // Reply (any DB depth): flat-append to its root's replies list.
  const rootId = reply.root_comment_id
  if (rootId == null) return
  const rootIdx = data.value.items.findIndex((c) => c.id === rootId)
  if (rootIdx < 0) return

  const nextItems = [...data.value.items]
  nextItems[rootIdx] = appendReplyToRoot(data.value.items[rootIdx]!, reply)
  data.value = { items: nextItems, total: data.value.total }
}

const handleReplyRemoved = (
  commentId: number,
  removedSize: number,
  rootId: number
) => {
  if (!data.value) return

  // Root removal: drop from data.items, decrement total, close drawer
  // if it was viewing the doomed root.
  const rootIdx = data.value.items.findIndex((c) => c.id === commentId)
  if (rootIdx >= 0) {
    const nextItems = [...data.value.items]
    nextItems.splice(rootIdx, 1)
    data.value = {
      items: nextItems,
      total: Math.max(0, (data.value.total ?? 0) - 1)
    }
    if (openThreadRootId.value === commentId) {
      openThreadRootId.value = null
    }
    return
  }

  // Reply removal: locate the owning root via rootId hint, then prune
  // the reply (and any loaded descendants of it) from the flat list.
  const ownerIdx = data.value.items.findIndex((c) => c.id === rootId)
  if (ownerIdx < 0) return

  const owner = data.value.items[ownerIdx]!
  const { node, pruned } = removeReplyFromRoot(owner, commentId, removedSize)
  const nextItems = [...data.value.items]
  nextItems[ownerIdx] = pruned
    ? node
    : {
        // Reply wasn't in the loaded slice (e.g. drawer deleted a
        // grandchild that never made it into the inline 3). Still
        // shrink the count so "查看更多 N" stays honest.
        ...owner,
        reply_count: Math.max(0, (owner.reply_count ?? 0) - removedSize)
      }
  data.value = { items: nextItems, total: data.value.total }
}

const handleReplyEdited = (updated: GalgameComment) => {
  if (!data.value) return
  const patch = updated as CommentNode
  const rootId = patch.root_comment_id ?? patch.id
  const rootIdx = data.value.items.findIndex((c) => c.id === rootId)
  if (rootIdx < 0) return

  const { node, replaced } = replaceCommentInRoot(
    data.value.items[rootIdx]!,
    patch
  )
  if (!replaced) return
  const nextItems = [...data.value.items]
  nextItems[rootIdx] = node
  data.value = { items: nextItems, total: data.value.total }
}

// ──────────────────────────────────────────
// Drawer
// ──────────────────────────────────────────

const openThreadRootId = ref<number | null>(null)
</script>

<template>
  <div class="space-y-3">
    <KunHeader name="游戏评论" scale="h2">
      <template #endContent>
        <KunLink size="sm" to="/topic/1482">
          Galgame 评论注意事项, 资源失效, 解压密码错误等问题反馈
        </KunLink>
      </template>
    </KunHeader>

    <div v-if="data" class="space-y-3">
      <GalgameCommentPanel @submitted="handleNewComment">
        <div v-if="data.total" class="flex items-center gap-2">
          <KunButton
            :is-icon-only="true"
            :variant="pageData.sort_order === 'desc' ? 'flat' : 'light'"
            size="lg"
            @click="pageData.sort_order = 'desc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-down" />
          </KunButton>

          <KunButton
            :is-icon-only="true"
            :variant="pageData.sort_order === 'asc' ? 'flat' : 'light'"
            size="lg"
            @click="pageData.sort_order = 'asc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-up" />
          </KunButton>
        </div>
      </GalgameCommentPanel>

      <KunLoading v-if="status === 'pending'" />

      <KunNull
        v-if="
          !data.total && status !== 'pending' && galgame?.is_on_forum !== false
        "
        description="没人评论, 是没人要这个 Galgame 的小只可爱软萌女孩子了吗, 呜呜呜呜呜呜！！"
      />

      <div v-if="status !== 'pending' && data.total" class="space-y-6">
        <GalgameComment
          v-for="comment in data.items"
          :key="comment.id"
          :comment="comment"
          :depth="0"
          @open-thread="(rootId) => (openThreadRootId = rootId)"
          @reply-added="handleNewComment"
          @reply-edited="handleReplyEdited"
          @reply-removed="handleReplyRemoved"
        />
      </div>

      <KunPagination
        v-if="data.total > 30 || data.total === 30"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <GalgameCommentThreadDrawer
      v-model:root-comment-id="openThreadRootId"
      :galgame-id="gid"
      @reply-added="handleNewComment"
      @reply-edited="handleReplyEdited"
      @reply-removed="handleReplyRemoved"
    />
  </div>
</template>
