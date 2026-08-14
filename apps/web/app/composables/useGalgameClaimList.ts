const CLAIM_PAGE_LIMIT = 20

export const useGalgameClaimList = async (path: string) => {
  const items = ref<UserClaimItem[]>([])
  const nextBefore = ref(0)
  const total = ref(0)
  const isLoadingMore = ref(false)

  const { data, refresh } = await useKunFetch<UserClaimList>(path, {
    query: { before: 0, limit: CLAIM_PAGE_LIMIT }
  })

  watch(
    data,
    (page) => {
      if (!page) {
        return
      }
      items.value = page.items
      nextBefore.value = page.next_before
      total.value = page.total
    },
    { immediate: true }
  )

  // The cursor alone is not a "there is more" signal: catalog keeps handing one
  // back on a short tail page, and following it fetched an empty page that
  // replaced the list with 什么都没有. The count is the authority.
  const hasMore = computed(
    () => nextBefore.value > 0 && items.value.length < total.value
  )

  const loadMore = async () => {
    if (isLoadingMore.value || !hasMore.value) {
      return
    }
    isLoadingMore.value = true
    const next = await kunFetch<UserClaimList>(path, {
      query: { before: nextBefore.value, limit: CLAIM_PAGE_LIMIT }
    })
    isLoadingMore.value = false
    if (!next) {
      return
    }
    items.value.push(...next.items)
    nextBefore.value = next.next_before
  }

  return { data, items, hasMore, isLoadingMore, loadMore, refresh }
}
