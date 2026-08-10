<script setup lang="ts">
defineProps<{ content: string; label?: string }>()
</script>

<template>
  <div class="text-default-500 flex gap-2 text-sm">
    <div class="bg-default-300 w-1 shrink-0 rounded-full" />
    <div class="min-w-0">
      <span v-if="label" class="text-default-400 mr-1 text-xs font-medium">
        {{ label }}
      </span>
      <span class="text-default-500 line-clamp-2 break-all">
        {{ markdownToText(content) }}
      </span>
    </div>
  </div>
</template>
