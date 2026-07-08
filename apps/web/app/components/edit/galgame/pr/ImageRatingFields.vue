<script setup lang="ts">
// The shared 性内容 / 暴力 rating selects for a cover or screenshot row
// (0=未评定, 1..3=轻/中/高 — the same 0..3 scheme the wiki + validation use).
// Exposed as two v-models so the parent binds the row's fields directly
// (v-model:sexual / v-model:violence); lives inside each tile's ⚙ popover so
// ratings are edited on demand, not as an always-expanded form per image.
const sexual = defineModel<number>('sexual', { default: 0 })
const violence = defineModel<number>('violence', { default: 0 })

const ratingOptions = [
  { value: 0, label: '未评定' },
  { value: 1, label: '轻' },
  { value: 2, label: '中' },
  { value: 3, label: '高' }
] as const
</script>

<template>
  <div class="space-y-2">
    <KunSelect v-model="sexual" label="性内容评级" :options="ratingOptions" />
    <KunSelect v-model="violence" label="暴力评级" :options="ratingOptions" />
  </div>
</template>
