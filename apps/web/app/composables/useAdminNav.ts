import {
  KUN_ADMIN_PAGE_ASIDE_NAV_ITEM,
  type KunAdminPageAsideItem
} from '~/constants/admin'

export const useAdminNav = () => {
  const { canAdminister } = useRole()
  const myPermissions = useMyPermissions()

  const items = computed(() =>
    KUN_ADMIN_PAGE_ASIDE_NAV_ITEM.filter((item) => {
      if (item.role === 'admin') {
        return canAdminister.value
      }
      return (
        !item.permissions ||
        item.permissions.some((key) => myPermissions.value(key))
      )
    })
  )

  const pathOf = (item: KunAdminPageAsideItem) =>
    item.to ?? `/admin/${item.router}`

  const entryPath = computed(() => {
    const first = items.value[0]
    return first ? pathOf(first) : null
  })

  return { items, entryPath, pathOf }
}
