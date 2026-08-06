import {
  KUN_ADMIN_PAGE_ASIDE_NAV_ITEM,
  type KunAdminPageAsideItem
} from '~/constants/admin'

// The 管理系统 rail, FILTERED to what this viewer can actually open — and the
// door that leads there.
//
// Both halves have to agree, which is why they live together. The rail was
// filtered per capability but the console's only entrance (the user menu) was a
// hard-coded link to /admin/overview, a page gated on `admin.dashboard`. A
// moderator holds no such key, so the guard bounced them straight back to the
// homepage: every console page was unreachable for them, including the ones
// they held the permission for. Granting a moderator update_log.create let them
// create a 待办 as far as the API was concerned, and they still could not get to
// the page to do it.
//
// So the entrance points at the FIRST entry the viewer may open, and hides
// itself when there is none. Anyone holding any console capability gets a door —
// including a roleless user carrying a single personal grant, who under the old
// `canModerate` check had none.
//
// UX only, as ever: each page keeps its own guard and the API is the boundary.
export const useAdminNav = () => {
  const { canModerate, canAdminister } = useRole()
  const myPermissions = useMyPermissions()

  const items = computed(() =>
    KUN_ADMIN_PAGE_ASIDE_NAV_ITEM.filter((item) => {
      if (item.role === 'admin') {
        return canAdminister.value
      }
      if (item.role === 'moderator') {
        return canModerate.value
      }
      // Any ONE of the listed keys is enough — 文档管理 is worth opening if you
      // can edit but not delete. An entry with neither field stays visible.
      return (
        !item.permissions ||
        item.permissions.some((key) => myPermissions.value(key))
      )
    })
  )

  // An entry outside /admin (更新日志与待办) carries its own absolute path.
  const pathOf = (item: KunAdminPageAsideItem) =>
    item.to ?? `/admin/${item.router}`

  const entryPath = computed(() => {
    const first = items.value[0]
    return first ? pathOf(first) : null
  })

  return { items, entryPath, pathOf }
}
