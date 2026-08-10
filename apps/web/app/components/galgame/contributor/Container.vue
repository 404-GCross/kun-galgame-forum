<script setup lang="ts">
const galgame = inject<GalgameDetail>('galgame')
const contributors = computed(() => galgame?.contributor ?? [])
</script>

<template>
  <div v-if="contributors.length" class="flex flex-wrap items-center gap-1">
    <KunTooltip
      v-for="user in contributors"
      :key="user.id"
      :text="user.name"
    >
      <KunAvatar :user="user" size="sm" />
    </KunTooltip>
  </div>
</template>
