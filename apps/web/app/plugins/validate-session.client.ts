export default defineNuxtPlugin((nuxtApp) => {
  const userStore = usePersistUserStore()
  if (!userStore.id) return

  nuxtApp.runWithContext(() => {
    kunFetch('/user/status')
    useRefreshMe().refreshMe()
  })
})
