const FLASH = ['outline-2', 'outline-offset-2', 'outline-primary', 'rounded-lg']

const scrollTo = (el: HTMLElement, flash: boolean) => {
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  if (!flash) return
  el.classList.add(...FLASH)
  setTimeout(() => el.classList.remove(...FLASH), 1500)
}

export const useTopicScroll = () => {
  const scrollToFloor = (floor: number, flash = true): boolean => {
    if (!import.meta.client || floor <= 0) return false
    const el = document.querySelector<HTMLElement>(`[id^="${floor}."]`)
    if (!el) return false
    scrollTo(el, flash)
    return true
  }

  const scrollToComment = (commentId: number, flash = true): boolean => {
    if (!import.meta.client || commentId <= 0) return false
    const el = document.getElementById(`comment-${commentId}`)
    if (!el) return false
    scrollTo(el, flash)
    return true
  }

  return { scrollToFloor, scrollToComment }
}
