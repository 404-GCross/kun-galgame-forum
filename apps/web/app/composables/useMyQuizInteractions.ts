export const useMyQuizInteractions = () => {
  const { id } = usePersistUserStore()
  const favorited = useState<number[]>('my-quiz-favorited', () => [])
  const loaded = useState<boolean>('my-quiz-interactions-loaded', () => false)

  const ensureLoaded = async () => {
    if (loaded.value || !id) return
    loaded.value = true
    const res = await kunFetch<{ favorited: number[] }>(
      '/galgame-quiz/mine/favorites'
    )
    if (res) {
      favorited.value = res.favorited ?? []
    } else {
      loaded.value = false
    }
  }

  const favoritedSet = computed(() => new Set(favorited.value))

  const setFavorited = (quizId: number, isFav: boolean) => {
    const set = new Set(favorited.value)
    if (isFav) {
      set.add(quizId)
    } else {
      set.delete(quizId)
    }
    favorited.value = [...set]
  }

  return {
    isFavorited: (quizId: number) => favoritedSet.value.has(quizId),
    setFavorited,
    ensureLoaded
  }
}
