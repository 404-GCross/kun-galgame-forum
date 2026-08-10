<script setup lang="ts">
defineProps<{
  actor?: KunUser
  timestamp: Date | string
}>()
</script>

<template>
  <div class="flex w-full gap-3">
    <KunAvatar v-if="actor" :user="actor" />

    <div class="min-w-0 flex-1 space-y-2">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
        <KunLink
          v-if="actor"
          :to="`/user/${actor.id}`"
          underline="none"
          color="default"
          class-name="text-default-800 hover:text-primary font-medium"
        >
          {{ actor.name }}
        </KunLink>
        <span class="text-default-500">
          <KunTime :time="timestamp" />
        </span>
        <slot name="meta" />
      </div>

      <slot />
    </div>
  </div>
</template>
