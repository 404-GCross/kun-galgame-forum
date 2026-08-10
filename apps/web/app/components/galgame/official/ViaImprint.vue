<script setup lang="ts">
defineProps<{
  name: string
}>()
</script>

<template>
  <p class="text-default-500 mt-1 flex items-center gap-1 text-xs">
    <KunIcon name="lucide:corner-down-right" class="shrink-0" />
    <span class="truncate">经 {{ name }}</span>
  </p>
</template>
