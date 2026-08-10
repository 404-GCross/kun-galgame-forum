export default defineNuxtPlugin(() => {
  const router = useRouter()

  document.addEventListener('click', (e) => {
    if (
      e.defaultPrevented ||
      e.button !== 0 ||
      e.metaKey ||
      e.ctrlKey ||
      e.shiftKey ||
      e.altKey
    ) {
      return
    }
    const mention = (e.target as HTMLElement | null)?.closest<HTMLElement>(
      'a.kun-mention'
    )
    const uid = mention?.dataset.uid
    if (!uid) {
      return
    }
    e.preventDefault()
    router.push(`/user/${uid}`)
  })
})
