<script setup lang="ts">
withDefaults(
  defineProps<{
    src: string
    name: string
    size?: 'md' | 'lg'
  }>(),
  { size: 'lg' }
)
</script>

<template>
  <div class="bg-default-100 shrink-0 rounded-lg p-2 sm:p-3">
    <KunImage
      :src="src"
      :alt="`${name} logo`"
      loading="eager"
      object-fit="contain"
      :class-name="
        cn(size === 'lg' ? 'size-14 sm:size-20' : 'size-10 sm:size-14')
      "
    />
  </div>
</template>
