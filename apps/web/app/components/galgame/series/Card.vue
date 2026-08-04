<script setup lang="ts">
defineProps<{
  series: GalgameSeriesItem
}>()
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="true"
    :href="`/galgame/series/${series.id}`"
  >
    <h3 class="text-default-900 font-semibold">
      {{ series.name }}
      <KunChip size="xs">
        {{ `+ ${series.galgame_count}` }}
      </KunChip>
    </h3>
  </KunCard>
</template>
