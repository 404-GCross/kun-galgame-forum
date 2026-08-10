export default defineNuxtPlugin(async () => {
  const mine = useState<string[] | null>('kun-perm-mine', () => null)

  if (mine.value) return

  const { id } = usePersistUserStore()
  if (!id) return

  const data = await kunFetch<KunPermMine>('/perm/mine')
  if (data) {
    mine.value = data.permissions
  }
})
