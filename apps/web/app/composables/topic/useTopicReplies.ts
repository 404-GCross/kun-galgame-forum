import type { Ref } from 'vue'

export const useTopicReplies = (topicId: number | Ref<number>) => {
  const _topicId = toValue(topicId)

  const replies = useState<TopicReply[]>(
    `kun-topic-replies-${_topicId}`,
    () => []
  )
  const isComplete = useState<boolean>(
    `kun-topic-replies-complete-${_topicId}`,
    () => false
  )

  const status = useState<'idle' | 'pending' | 'success' | 'error'>(
    `kun-topic-replies-status-${_topicId}`,
    () => 'idle'
  )

  const minPage = useState<number>(`kun-topic-replies-min-${_topicId}`, () => 1)
  const maxPage = useState<number>(`kun-topic-replies-max-${_topicId}`, () => 1)
  const sortOrder = useState<'asc' | 'desc'>(
    `kun-topic-replies-sort-${_topicId}`,
    () => 'asc'
  )

  const hasEarlier = computed(() => minPage.value > 1)

  const _fetchReplies = async (
    fetchPage: number,
    fetchSortOrder: 'asc' | 'desc'
  ) => {
    status.value = 'pending'

    const newReplies = await kunFetch<TopicReply[]>(
      `/topic/${_topicId}/reply`,
      {
        query: {
          topic_id: _topicId,
          page: fetchPage,
          limit: 30,
          sort_order: fetchSortOrder
        }
      }
    )
    status.value = 'success'
    return newReplies ?? []
  }

  const loadInitialReplies = async (startPage = 1) => {
    if (replies.value.length > 0) {
      return
    }

    const page = Math.max(1, startPage)
    sortOrder.value = 'asc'
    minPage.value = page
    maxPage.value = page

    const data = await _fetchReplies(page, sortOrder.value)
    isComplete.value = data.length < 30
    replies.value = data
  }

  const loadMore = async () => {
    if (status.value === 'pending' || isComplete.value) return

    const next = maxPage.value + 1
    const newReplies = await _fetchReplies(next, sortOrder.value)
    maxPage.value = next
    if (newReplies.length < 30) {
      isComplete.value = true
    }
    replies.value.push(...newReplies)
  }

  const loadEarlier = async () => {
    if (status.value === 'pending' || minPage.value <= 1) return

    const prev = minPage.value - 1
    const newReplies = await _fetchReplies(prev, sortOrder.value)
    minPage.value = prev
    replies.value.unshift(...newReplies)
  }

  const setSort = async (order: 'asc' | 'desc') => {
    if (status.value === 'pending' || sortOrder.value === order) return

    sortOrder.value = order
    minPage.value = 1
    maxPage.value = 1
    isComplete.value = false

    const sortedReplies = await _fetchReplies(1, sortOrder.value)
    if (sortedReplies.length < 30) {
      isComplete.value = true
    }
    replies.value = sortedReplies
  }

  const addNewReply = (newReply: TopicReply) => {
    if (replies.value.some((r) => r.id === newReply.id)) return

    if (sortOrder.value === 'desc' && minPage.value === 1) {
      replies.value.unshift(newReply)
    } else {
      replies.value.push(newReply)
    }
  }

  const updateReply = (updatedReply: TopicReply) => {
    const index = replies.value.findIndex((r) => r.id === updatedReply.id)
    if (index !== -1) {
      replies.value[index] = updatedReply
    }
  }

  const removeReply = (replyId: number) => {
    const index = replies.value.findIndex((r) => r.id === replyId)
    if (index !== -1) {
      replies.value.splice(index, 1)
    }
  }

  return {
    replies,
    status,
    isComplete,
    hasEarlier,
    sortOrder,
    loadInitialReplies,
    loadMore,
    loadEarlier,
    setSort,
    addNewReply,
    updateReply,
    removeReply
  }
}
