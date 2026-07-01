<script setup lang="ts">
// Galgame submission review queue. Lists messages from the wiki
// /admin/galgame/messages endpoint (proxied through kungal) and exposes
// approve / decline / ban actions via PUT /api/admin/galgame/:gid/status.
//
// See docs/galgame_wiki/06-admin.md §PUT /admin/galgame/:gid/status for
// the upstream contract and 08-messages.md for the message shape.

useKunDisableSeo('Galgame 审核')

interface AdminQueueGalgame {
  id: number
  name_zh_cn?: string
  name_ja_jp?: string
  name_en_us?: string
  name_zh_tw?: string
  banner?: string
  // K-PR6: banner_image_hash retired in wiki PR5. walker injects
  // effective_banner_url on every wiki object that carries
  // effective_banner_hash, including embeddeds in wiki messages.
  effective_banner_hash?: string
  effective_banner_url?: string
  status: number
  user_id?: number
}

interface AdminQueueActor {
  id: number
  name: string
  avatar: string
}

interface AdminQueueMessage {
  id: number
  type: string
  galgame_id: number
  galgame: AdminQueueGalgame | null
  actor_user_id: number
  actor?: AdminQueueActor
  payload: Record<string, unknown> | null
  created_at: string
}

interface AdminQueueEnvelope {
  items: AdminQueueMessage[]
  total: number
}

// Default filter: submitted + edited_pending (both are "needs review"
// triggers). admin can toggle whichever they want to drill into.
const pageData = reactive({
  type: 'submitted,edited_pending',
  page: 1,
  limit: 20
})

const { data, status, refresh } = await useKunFetch<AdminQueueEnvelope>(
  '/admin/galgame/messages',
  { query: pageData }
)

// Wire-name resolution is shared (shared/utils/galgameStatus.ts).
// typeBadge stays local: the admin queue wants queue-specific wording
// ("新提交" / "修订重审") distinct from the user-facing notification
// wording in the message center, so this is intentionally NOT shared.
// Row title. Prefer the galgame's localized name; for a brand-new submission
// whose names aren't imported yet (name_* all empty), fall back to the VNDB id
// carried in the `submitted` payload so the row stays identifiable, then to
// #id. (The message brief itself carries no vndb_id — only the payload does.)
// Once infra ships reviewer-visible drafts, 查看 opens the full detail anyway.
const displayName = (msg: AdminQueueMessage): string => {
  const named = msg.galgame ? galgameNameFromWire(msg.galgame, '') : ''
  if (named) return named
  const vndb =
    msg.payload && typeof msg.payload.vndb_id === 'string'
      ? msg.payload.vndb_id
      : ''
  if (vndb) return `VNDB ${vndb}`
  return msg.galgame ? `#${msg.galgame.id}` : `#${msg.galgame_id}`
}

const typeBadge = (t: string) => {
  switch (t) {
    case 'submitted':
      return { label: '新提交', color: 'primary' as const }
    case 'edited_pending':
      return { label: '修订重审', color: 'warning' as const }
    default:
      return { label: t, color: 'default' as const }
  }
}

const filterTabs = [
  { value: 'submitted,edited_pending', label: '全部' },
  { value: 'submitted', label: '仅新提交' },
  { value: 'edited_pending', label: '仅修订重审' }
] as const

const isActing = ref<Record<number, boolean>>({})

// Reason-capture modal state. KunUI doesn't have a built-in prompt
// dialog and useComponentMessageStore only exposes a confirm-style
// `alert`, so we inline a small KunModal + KunTextarea here for the
// two flows that need a free-text reason (decline / ban).
type ReasonAction = 'decline' | 'ban'

interface ReasonContext {
  action: ReasonAction
  target: AdminQueueMessage
  status: number
}

const isReasonModalOpen = ref(false)
const reasonContext = ref<ReasonContext | null>(null)
const reasonText = ref('')

const modalTitle = computed(() => {
  if (!reasonContext.value) return ''
  const name = displayName(reasonContext.value.target)
  return reasonContext.value.action === 'decline'
    ? `拒绝《${name}》`
    : `封禁《${name}》`
})

const modalDescription = computed(() => {
  if (!reasonContext.value) return ''
  return reasonContext.value.action === 'decline'
    ? '请填写拒绝原因, 提交者会在站内消息中看到。'
    : '封禁后该条目对所有人不可见。如不填写理由, 仅记录管理员操作。'
})

const modalRequiresReason = computed(
  () => reasonContext.value?.action === 'decline'
)

const openReasonModal = (action: ReasonAction, target: AdminQueueMessage) => {
  reasonContext.value = {
    action,
    target,
    status: action === 'decline' ? 4 : 1
  }
  reasonText.value = ''
  isReasonModalOpen.value = true
}

const closeReasonModal = () => {
  isReasonModalOpen.value = false
  // Clear the context after the leave-transition so the title/desc
  // computeds don't blink to empty mid-animation.
  setTimeout(() => {
    reasonContext.value = null
    reasonText.value = ''
  }, 300)
}

const applyStatus = async (
  gid: number,
  msgId: number,
  target: number,
  reason: string
) => {
  // PUT /api/admin/galgame/:gid/status — kungal forwards to wiki with the
  // admin's Bearer attached so wiki's revision/message audit records the
  // right actor.
  isActing.value = { ...isActing.value, [msgId]: true }
  const res = await kunFetch<unknown>(`/admin/galgame/${gid}/status`, {
    method: 'PUT',
    body: { status: target, reason }
  })
  isActing.value = { ...isActing.value, [msgId]: false }
  if (res !== null) {
    useMessage('已处理', 'success')
    refresh()
  }
}

const handleApprove = async (msg: AdminQueueMessage) => {
  const ok = await useComponentMessageStore().alert(
    `通过《${displayName(msg)}》?`,
    '通过后该 Galgame 立即公开发布, 提交者会收到通知并自动获得 +3 萌萌点 (通过 wiki cron 同步)。'
  )
  if (!ok) return
  await applyStatus(msg.galgame_id, msg.id, 0, '')
}

const handleConfirmReason = async () => {
  const ctx = reasonContext.value
  if (!ctx) return
  const trimmed = reasonText.value.trim()
  if (modalRequiresReason.value && !trimmed) {
    useMessage('拒绝原因不能为空', 'warn')
    return
  }
  const msgId = ctx.target.id
  closeReasonModal()
  await applyStatus(ctx.target.galgame_id, msgId, ctx.status, trimmed)
}
</script>

<template>
  <div v-if="data" class="w-full space-y-4">
    <KunHeader
      name="Galgame 审核"
      description="审核用户提交的新 Galgame 及对自己草稿的修订。通过后立即公开发布并向提交者发放 +3 萌萌点; 拒绝时需说明原因, 提交者可据此修改后重新提交。"
    >
      <template #endContent>
        <div class="flex flex-wrap gap-2">
          <KunButton
            v-for="tab in filterTabs"
            :key="tab.value"
            size="sm"
            :variant="pageData.type === tab.value ? 'flat' : 'light'"
            @click="pageData.type = tab.value"
          >
            {{ tab.label }}
          </KunButton>
        </div>
      </template>
    </KunHeader>

    <KunDivider />

    <div v-if="data.items.length" class="flex flex-col gap-3">
      <div
        v-for="msg in data.items"
        :key="msg.id"
        class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-start"
      >
        <KunImage
          v-if="getEffectiveBanner(msg.galgame)"
          :src="getEffectiveBanner(msg.galgame, { variant: 'mini' })"
          loading="lazy"
          placeholder="/placeholder.webp"
          class="h-20 w-32 shrink-0 rounded object-cover"
          :style="{ aspectRatio: '16/9' }"
        />
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="truncate text-lg font-medium">
              {{ displayName(msg) }}
            </h3>
            <KunChip size="xs" variant="flat" :color="typeBadge(msg.type).color">
              {{ typeBadge(msg.type).label }}
            </KunChip>
          </div>
          <div class="text-default-500 flex flex-wrap items-center gap-2 text-sm">
            <span>
              提交者:
              <KunLink v-if="msg.actor" :to="`/user/${msg.actor.id}`">
                {{ msg.actor.name }}
              </KunLink>
              <template v-else>#{{ msg.actor_user_id }}</template>
            </span>
            <span>·</span>
            <span><KunTime :time="msg.created_at" /></span>
            <span>·</span>
            <span>galgame_id: {{ msg.galgame_id }}</span>
          </div>
        </div>
        <div class="flex shrink-0 flex-wrap gap-2">
          <KunLink
            v-if="msg.galgame"
            :to="`/galgame/${msg.galgame.id}`"
            target="_blank"
          >
            <KunButton size="sm" variant="flat">查看</KunButton>
          </KunLink>
          <KunButton
            size="sm"
            color="success"
            :loading="isActing[msg.id]"
            :disabled="isActing[msg.id]"
            @click="handleApprove(msg)"
          >
            通过
          </KunButton>
          <KunButton
            size="sm"
            color="danger"
            variant="flat"
            :loading="isActing[msg.id]"
            :disabled="isActing[msg.id]"
            @click="openReasonModal('decline', msg)"
          >
            拒绝
          </KunButton>
          <KunButton
            size="sm"
            color="default"
            variant="light"
            :loading="isActing[msg.id]"
            :disabled="isActing[msg.id]"
            @click="openReasonModal('ban', msg)"
          >
            封禁
          </KunButton>
        </div>
      </div>
    </div>

    <KunNull v-if="!data.total" />

    <KunPagination
      v-if="data.total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <KunModal
      :model-value="isReasonModalOpen"
      inner-class-name="w-full max-w-lg"
      :is-dismissable="false"
      @update:model-value="closeReasonModal"
    >
      <div class="space-y-4">
        <h3 class="text-xl font-medium">{{ modalTitle }}</h3>
        <p class="text-default-500 text-sm">{{ modalDescription }}</p>

        <KunTextarea
          v-model="reasonText"
          :placeholder="
            modalRequiresReason
              ? '请填写拒绝原因 (必填)'
              : '可选填理由'
          "
          :rows="4"
          :maxlength="1007"
          show-char-count
          :required="modalRequiresReason"
        />

        <div class="flex justify-end gap-2">
          <KunButton variant="light" @click="closeReasonModal">取消</KunButton>
          <KunButton
            :color="
              reasonContext?.action === 'decline' ? 'danger' : 'default'
            "
            @click="handleConfirmReason"
          >
            确认
          </KunButton>
        </div>
      </div>
    </KunModal>
  </div>
</template>
