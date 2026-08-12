export default defineNuxtPlugin((nuxtApp) => {
  const { refreshMe } = useRefreshMe()

  const onVisible = () => {
    if (document.visibilityState === 'visible') {
      nuxtApp.runWithContext(() => refreshMe())
    }
  }
  const onOnline = () => nuxtApp.runWithContext(() => refreshMe())

  document.addEventListener('visibilitychange', onVisible)
  window.addEventListener('online', onOnline)
})
