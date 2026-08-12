export interface UpvoteTarget {
  topicId: number
  targetUserId: number
}

const isOpen = ref(false)
const target = ref<UpvoteTarget | null>(null)
let settle: ((pushed: boolean) => void) | null = null

export const useUpvoteModal = () => {
  const open = (t: UpvoteTarget) =>
    new Promise<boolean>((resolve) => {
      target.value = t
      settle = resolve
      isOpen.value = true
    })

  const close = (pushed: boolean) => {
    const resolve = settle
    settle = null
    isOpen.value = false
    resolve?.(pushed)
  }

  return { isOpen, target, open, close }
}
