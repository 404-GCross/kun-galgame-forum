import type { ForumPermission } from '~/composables/useCan'
import { KUN_PERMISSION_KEYS } from '~/constants/permission'

// The `permissions` page-meta key this guard reads. Declared HERE, next to its
// only consumer, and deliberately NOT in shared/nuxt.d.ts: that file belongs to
// the shared/server TS project, and importing an app composable into it drags
// useCan.ts across the project boundary, where Nuxt's auto-imports don't exist
// and every `computed` / `useState` in it stops resolving.
//
// This buys editor completion, NOT safety: `definePageMeta`'s PageMeta carries a
// `[key: string]: unknown` index signature, so a misspelled key still typechecks
// (verified — vue-tsc passes on `'friend_link.creat'`). Since an unknown key
// matches nobody, the guard would fail closed and lock EVERY viewer out of the
// page, silently. Hence the runtime assertion below.
declare module 'vue-router' {
  interface RouteMeta {
    // The page opens for anyone holding ANY ONE of these keys.
    permissions?: ForumPermission[]
  }
}

// Capability route guard: allows the route if the viewer holds ANY ONE of the
// keys the page declares.
//
//   definePageMeta({ middleware: 'permission', permissions: ['doc.edit'] })
//
// Prefer this over the `moderator` / `admin` tier guards for any page whose API
// routes are gated with RequirePermission. The tier guards ask a DIFFERENT
// question than the backend does, and the two answers come apart the moment an
// override is in play: 友链管理 and 文档管理 sit behind `moderator`, but their
// endpoints check friend_link.* / doc.*, all of which are currently revoked from
// the moderator role in production. A moderator therefore walked past the tier
// guard onto a page where the very first list call 403s. Guarding on the same
// key the API checks keeps the UX mirror honest.
//
// UX only, as ever — pkg/perm with both override layers applied is the boundary.
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

  // A key that is not in the vocabulary can never match, so the guard below
  // would turn it into a page nobody may open — the kind of failure that looks
  // like a permission mysteriously disappearing. Say so instead of swallowing it.
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
