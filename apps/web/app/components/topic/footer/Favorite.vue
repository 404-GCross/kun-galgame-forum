<script setup lang="ts">
const props = defineProps<{
  topicId: number
  favoriteCount: number
  isFavorite: boolean
}>()
</script>

<template>
  <FavoriteToggle
    :favorited="props.isFavorite"
    :count="props.favoriteCount"
    :endpoint="`/topic/${props.topicId}/favorite`"
    :messages="[10230, 10231]"
  />
</template>
