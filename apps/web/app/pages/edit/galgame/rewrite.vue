<script setup lang="ts">
definePageMeta({
  middleware: ['auth', 'prevent']
})

useKunDisableSeo('重新编辑 Galgame')
</script>

<template>
  <EditGalgamePrPullRequest />
</template>
