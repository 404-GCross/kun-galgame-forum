<script setup lang="ts">
// /admin has no content of its own — it forwards to the first console surface
// this viewer may open, so a bookmark or a typed URL behaves like clicking
// 管理系统 in the user menu. Without it the route rendered the rail beside an
// empty <NuxtPage>, which reads as a broken console.
//
// The redirect runs in middleware rather than setup so it resolves before the
// rail paints (no flash of an empty page), and covers SSR as well.
definePageMeta({
  middleware: () => {
    const { id } = usePersistUserStore()
    if (!id) {
      return navigateTo('/auth/required?redirect=%2Fadmin')
    }
    // Nobody holding any console capability — the same place every page guard
    // sends a viewer it turns away.
    const { entryPath } = useAdminNav()
    return navigateTo(entryPath.value ?? '/')
  }
})
</script>

<template>
  <div />
</template>
