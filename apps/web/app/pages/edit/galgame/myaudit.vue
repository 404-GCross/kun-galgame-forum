<script setup lang="ts">
definePageMeta({
  middleware: ['moderator']
})

useKunDisableSeo('我的 Galgame 审核')
</script>

<template>
  <EditGalgameMineAudit />
</template>
