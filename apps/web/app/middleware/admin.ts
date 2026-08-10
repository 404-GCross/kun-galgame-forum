export default defineNuxtRouteMiddleware((to) => {
  const { id } = usePersistUserStore()

  if (!id) {
    return navigateTo(
      `/auth/required?redirect=${encodeURIComponent(to.fullPath)}`
    )
  }

  const { canAdminister } = useRole()
  if (!canAdminister.value) {
    return navigateTo('/')
  }
})
