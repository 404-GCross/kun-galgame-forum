<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    loading?: boolean
    delay?: number
  }>(),
  { delay: 200 }
)
</script>

<template>
  <div
    :class="['kun-loading-dim', { 'kun-loading-dim--busy': loading }]"
    :style="{ '--kun-dim-delay': `${props.delay}ms` }"
    :inert="loading || undefined"
    :aria-busy="loading || undefined"
  >
    <slot />
  </div>
</template>

<style scoped>
.kun-loading-dim {
  opacity: 1;
}
.kun-loading-dim--busy {
  opacity: 0.5;
  transition: opacity 0.2s var(--kun-dim-delay, 200ms) linear;
}
@media (prefers-reduced-motion: reduce) {
  .kun-loading-dim--busy {
    transition: none;
  }
}
</style>
