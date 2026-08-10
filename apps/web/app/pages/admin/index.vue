<script setup lang="ts">
definePageMeta({
  middleware: () => {
    const { id } = usePersistUserStore()
    if (!id) {
      return navigateTo('/auth/required?redirect=%2Fadmin')
    }
    const { entryPath } = useAdminNav()
    return navigateTo(entryPath.value ?? '/')
  }
})
</script>

<template>
  <div />
</template>
