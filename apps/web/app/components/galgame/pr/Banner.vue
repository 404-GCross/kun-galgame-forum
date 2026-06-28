<script setup lang="ts">
// Creator/admin-only banner surfacing pending update requests (PRs) at the top
// of the galgame detail page. The PR review UI already exists but is buried
// behind the info-card "编辑历史 / 更新请求" modal, so owners never noticed
// pending edits. This pulls the pending count up front and deep-links into the
// existing modal's 更新请求 tab via the shared `galgameActivity` controller.
//
// Client-only + gated: the persist user store is only reliable after hydration
// and reviewers are a tiny audience, so the PR list fetch never runs on SSR or
// for anonymous / non-reviewer visitors (server:false + immediate gate) — zero
// SEO impact, no wasted requests.
const props = defineProps<{
  galgame: GalgameDetail
}>()

const { id } = usePersistUserStore()
const { canModerate } = useRole()
// Same gate as pr/Details.vue's merge/decline buttons: the galgame creator or
// any admin/moderator (canModerate) — i.e. exactly who can act on a PR.
const canReview = computed(
  () => props.galgame.user.id === id || canModerate.value
)

const activity = inject<{ open: boolean; tab: 'history' | 'pr' }>(
  'galgameActivity'
)

const { data, refresh } = await useKunFetch<{
  items: GalgamePR[]
  total: number
}>(`/galgame/${props.galgame.id}/pr/all`, {
  lazy: true,
  method: 'GET',
  server: false,
  immediate: canReview.value,
  query: { page: 1, limit: 50, galgameId: props.galgame.id }
})

const pendingCount = computed(() => {
  const items = data.value?.items ?? []
  return items.filter((pr) => pr.status === 0).length
})

// Re-check after the review modal closes: the user may have just merged or
// declined the pending PRs from inside it, so the count (and thus the banner)
// must self-correct without a full page reload.
watch(
  () => activity?.open,
  (isOpen, wasOpen) => {
    if (wasOpen && !isOpen && canReview.value) {
      refresh()
    }
  }
)

const openPr = () => {
  if (!activity) {
    return
  }
  activity.tab = 'pr'
  activity.open = true
}
</script>

<template>
  <button
    v-if="canReview && pendingCount > 0"
    type="button"
    class="bg-warning/15 text-warning hover:bg-warning/25 flex w-full items-center gap-2 rounded-lg p-3 text-left text-sm font-medium transition-colors"
    @click="openPr"
  >
    <KunIcon name="lucide:git-pull-request" class="shrink-0" />
    <span>
      该 Galgame 存在 {{ pendingCount }} 条需要审核的更新请求, 点击查看
    </span>
  </button>
</template>
