export const useMyGalgameInteractions = () => {
  const { id } = usePersistUserStore()
  const liked = useState<number[]>('my-galgame-liked', () => [])
  const favorited = useState<number[]>('my-galgame-favorited', () => [])
  const loaded = useState<boolean>(
    'my-galgame-interactions-loaded',
    () => false
  )

  const ensureLoaded = async () => {
    if (loaded.value || !id) return
    loaded.value = true
    const res = await kunFetch<{ liked: number[]; favorited: number[] }>(
      '/galgame/interactions/mine'
    )
    if (res) {
      liked.value = res.liked ?? []
      favorited.value = res.favorited ?? []
    } else {
      loaded.value = false
    }
  }

  const likedSet = computed(() => new Set(liked.value))
  const favoritedSet = computed(() => new Set(favorited.value))

  const setFavorited = (gid: number, isFav: boolean) => {
    const set = new Set(favorited.value)
    if (isFav) {
      set.add(gid)
    } else {
      set.delete(gid)
    }
    favorited.value = [...set]
  }

  return {
    isLiked: (gid: number) => likedSet.value.has(gid),
    isFavorited: (gid: number) => favoritedSet.value.has(gid),
    setFavorited,
    ensureLoaded
  }
}
