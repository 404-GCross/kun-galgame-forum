<script setup lang="ts">
// Galgame notification list. Data source is GET /api/galgame/messages/mine,
// which proxies the galgame service. After viewing, we advance the
// per-user read marker via PUT /api/galgame/messages/read-state so the
// aside badge clears.
//
// Why this isn't /message/notice (the local notice table):
//   - galgame messages live in the galgame service's own galgame_message table
//     (see docs/galgame_wiki/08-messages.md). Kungal doesn't mirror them
//     into its local message table — dual-writes are a known anti-pattern.
//   - Read state is per-consumer (kungal vs moyu vs admin UI), so it sits
//     in the kungal-local wiki_message_read_state table, not in galgame.

import type { GalgameMessageItem } from '~/components/message/aside/Galgame.vue'

definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('资料库通知')

interface GalgameMessagesEnvelope {
  items: GalgameMessageItem[]
  total: number
}

const pageData = reactive({
  page: 1,
  limit: 30
})

// Galgame returns id-desc; the page param here is for kungal-side
// pagination across long histories.
const { data, status } = await useKunFetch<GalgameMessagesEnvelope>(
  '/galgame/messages/mine',
  {
    query: computed(() => ({
      since_id: 0,
      limit: pageData.limit,
      page: pageData.page
    }))
  }
)

// Advance the read marker after the user lands on this page. We compute
// the highest id from the current page (server returns id-desc; items[0]
// is the latest) and best-effort write it — failures are non-fatal,
// they'll just leave the badge showing one extra time.
onMounted(async () => {
  const top = data.value?.items?.[0]
  if (!top) return
  await kunFetch('/galgame/messages/read-state', {
    method: 'PUT',
    body: { last_read_message_id: top.id }
  })
})
</script>

<template>
  <div class="flex w-full flex-col space-y-3" v-if="data">
    <header class="flex items-center gap-2">
      <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
        <KunIcon name="lucide:chevron-left" />
      </KunButton>
      <h2 class="text-lg">资料库通知</h2>
    </header>

    <KunDivider />

    <KunOverlayScroll v-if="data.items.length" class="h-full">
      <MessageAsideGalgame v-for="msg in data.items" :key="msg.id" :message="msg" />
    </KunOverlayScroll>

    <KunNull v-if="!data.total" />

    <KunPagination
      v-if="data.total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
