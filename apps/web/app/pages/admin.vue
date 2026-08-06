<script setup lang="ts">
import {
  KUN_ADMIN_PAGE_ASIDE_NAV_ITEM,
  type KUN_ADMIN_PAGE_ROUTE_TYPE
} from '~/constants/admin'

useKunDisableSeo('管理系统')

const route = useRoute()
const pageType = computed(() => {
  const routeType = route.fullPath.split('/').pop()
  return routeType as KUN_ADMIN_PAGE_ROUTE_TYPE
})

// Underlined vertical tab rail (same style as the home feed, one size up).
// Selecting a tab navigates to /admin/<router>; the active tab tracks the route.
//
// The rail is FILTERED to what this viewer can actually open. It used to render
// the full list unconditionally, so a moderator saw 数据总览 / 用户管理 / 权限管理
// / 网站设置 — four tabs whose pages carry `middleware: 'admin'` and bounce them
// straight back to the homepage. A tab you are shown and then thrown out of
// reads as the site being broken, or as a permission having gone missing.
const { canModerate, canAdminister } = useRole()
const myPermissions = useMyPermissions()

const adminNavItems = computed(() =>
  KUN_ADMIN_PAGE_ASIDE_NAV_ITEM.filter((item) => {
    if (item.role === 'admin') {
      return canAdminister.value
    }
    if (item.role === 'moderator') {
      return canModerate.value
    }
    // Any ONE of the listed keys is enough — 文档管理 is worth opening if you can
    // edit but not delete. An entry with neither field declared stays visible.
    return (
      !item.permissions ||
      item.permissions.some((key) => myPermissions.value(key))
    )
  }).map((item) => ({
    value: item.to ?? item.router!,
    textValue: item.label,
    icon: item.icon
  }))
)
</script>

<template>
  <div class="flex gap-3">
    <div class="hidden w-48 shrink-0 sm:block">
      <KunTab
        :model-value="pageType"
        :items="adminNavItems"
        orientation="vertical"
        variant="underlined"
        color="primary"
        size="lg"
        full-width
        @update:model-value="
          (value) =>
            navigateTo(value.startsWith('/') ? value : `/admin/${value}`)
        "
      />
    </div>

    <NuxtPage />
  </div>
</template>
