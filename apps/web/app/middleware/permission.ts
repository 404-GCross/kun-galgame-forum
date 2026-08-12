import type { ForumPermission } from '~/composables/useCan'
import { KUN_PERMISSION_KEYS } from '~/constants/permission'

declare module 'vue-router' {
  interface RouteMeta {
    permissions?: ForumPermission[]
  }
}

export default defineNuxtRouteMiddleware((to) => {
  const { id } = usePersistUserStore()

  if (!id) {
    return navigateTo(
      `/auth/required?redirect=${encodeURIComponent(to.fullPath)}`
    )
  }

  const required = (to.meta.permissions ?? []) as ForumPermission[]
  if (!required.length) {
    return
  }

  const unknown = required.filter((key) => !KUN_PERMISSION_KEYS.includes(key))
  if (unknown.length) {
    console.error(
      `[permission] ${to.fullPath} requires unknown permission key(s): ${unknown.join(', ')}`
    )
  }

  const myPermissions = useMyPermissions()
  if (!required.some((key) => myPermissions.value(key))) {
    return navigateTo('/')
  }
})
