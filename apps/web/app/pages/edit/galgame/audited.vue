<script setup lang="ts">
definePageMeta({
  middleware: 'permission',
  permissions: ['galgame.claim.review']
})

useKunDisableSeo('我的 Galgame 审核')
</script>

<template>
  <EditGalgameAudited />
</template>
