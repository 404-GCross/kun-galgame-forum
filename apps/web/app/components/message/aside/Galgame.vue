<script setup lang="ts">
// Single galgame-message item, modeled after MessageAsideNotice.
//
// Galgame messages have a richer shape than kungal's local notice rows:
// the target context is a galgame (not a forum reply / like / etc.) and
// the action is admin-triggered moderation rather than peer interaction.
// We translate that into a notice-style layout so the message center
// feels homogeneous across data sources.
//
// See docs/galgame_wiki/08-messages.md for the upstream payload schema.

interface GalgameMessageEntry {
  id: number
  name_zh_cn?: string
  name_ja_jp?: string
  name_en_us?: string
  name_zh_tw?: string
  // K-PR6: banner_image_hash retired in galgame PR5; effective_banner_hash
  // is the derived banner source (= covers[sort_order=0].image_hash).
  effective_banner_hash?: string
  status: number
}

export interface GalgameMessageItem {
  id: number
  type: string
  galgame_id: number
  galgame: GalgameMessageEntry | null
  actor_user_id: number
  target_user_id: number | null
  payload: Record<string, unknown> | null
  created_at: string
}

const props = defineProps<{
  message: GalgameMessageItem
}>()

const typeBadge = computed(() => {
  switch (props.message.type) {
    case 'approved':
      return { label: '审核通过', color: 'success' as const }
    case 'declined':
      return { label: '审核拒绝', color: 'danger' as const }
    case 'banned':
      return { label: '已封禁', color: 'default' as const }
    case 'unbanned':
      return { label: '已恢复', color: 'primary' as const }
    case 'submitted':
      return { label: '已提交', color: 'primary' as const }
    case 'claimed':
      return { label: '已认领', color: 'success' as const }
    case 'edited_pending':
      return { label: '重新审核', color: 'warning' as const }
    default:
      return { label: props.message.type, color: 'default' as const }
  }
})

// Wire-name resolution is shared (shared/utils/galgameStatus.ts). The
// message typeBadge stays local — notification wording ("审核通过") is
// intentionally distinct from the admin queue's wording.
const galgameName = computed(() => {
  const g = props.message.galgame
  if (!g) return '(已删除)'
  return galgameNameFromWire(g, `#${g.id}`)
})

const blurb = computed(() => {
  const name = galgameName.value
  switch (props.message.type) {
    case 'approved':
      return `您提交的《${name}》已通过审核, 萌萌点 +3 已发放。`
    case 'declined': {
      // Decline messages may carry a `reason` string in payload (set by
      // admin via the review queue). Surface it inline so the submitter
      // can act on the feedback without leaving the message center.
      const reasonRaw =
        props.message.payload && typeof props.message.payload.reason === 'string'
          ? String(props.message.payload.reason)
          : ''
      const reason = reasonRaw.trim()
      return reason
        ? `您提交的《${name}》被拒绝: ${reason}。您可以在「我的提交」继续编辑后重新提交。`
        : `您提交的《${name}》被拒绝。您可以在「我的提交」继续编辑后重新提交。`
    }
    case 'banned':
      return `您参与的《${name}》已被封禁。`
    case 'unbanned':
      return `《${name}》已恢复发布。`
    default:
      return `《${name}》: ${props.message.type}`
  }
})

const linkTo = computed(() => {
  const g = props.message.galgame
  if (!g) return '/edit/galgame/mine'
  if (g.status === GalgameStatus.Published) {
    return `/galgame/${g.id}`
  }
  return '/edit/galgame/mine'
})
</script>

<template>
  <div class="space-y-2 rounded-lg p-2">
    <div class="flex items-center gap-2 break-all">
      <KunIcon class="text-secondary text-lg" name="lucide:info" />
      <KunChip size="xs" variant="flat" :color="typeBadge.color">
        {{ typeBadge.label }}
      </KunChip>
      <span class="text-default-500 text-sm">
        <KunTime :time="message.created_at" type="datetime" show-year />
      </span>
    </div>

    <KunLink
      color="default"
      underline="none"
      :to="linkTo"
      class="hover:text-primary block cursor-pointer transition-colors"
    >
      <pre class="break-word text-sm leading-8 whitespace-pre-line text-inherit">{{ blurb }}</pre>
    </KunLink>
  </div>
</template>
