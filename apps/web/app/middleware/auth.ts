export default defineNuxtRouteMiddleware((to) => {
  const { id } = usePersistUserStore()

  if (!id) {
    return navigateTo(
      `/auth/required?redirect=${encodeURIComponent(to.fullPath)}`
    )
  }
})
