<script setup lang="ts">
definePageMeta({
  middleware: ['auth']
})

useKunDisableSeo('编辑草稿')
</script>

<template>
  <EditGalgameDraft />
</template>
