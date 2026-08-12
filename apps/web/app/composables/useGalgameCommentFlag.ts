const isOpen = ref(false)
const targetPostId = ref<number | null>(null)

export const useGalgameCommentFlag = () => {
  const open = (postId: number) => {
    targetPostId.value = postId
    isOpen.value = true
  }
  return { isOpen, targetPostId, open }
}
