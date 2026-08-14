<script setup lang="ts">
defineProps<{
  item: UserClaimItem
  timeLabel: string
}>()
</script>

<template>
  <div
    class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
  >
    <div class="min-w-0 flex-1 space-y-1">
      <div class="flex flex-wrap items-center gap-2">
        <h3
          class="hover:text-primary truncate text-lg font-medium transition-colors"
        >
          {{ item.display_name || '(无标题)' }}
        </h3>
        <KunChip
          size="xs"
          variant="flat"
          :color="galgameClaimStateBadge(item.claim_state).color"
        >
          {{ galgameClaimStateBadge(item.claim_state).label }}
        </KunChip>
      </div>

      <div class="text-default-500 flex flex-wrap items-center gap-2 text-sm">
        <span>{{ timeLabel }} <KunTime :time="item.first_acted_at" /></span>
        <template v-if="item.last_event_at !== item.first_acted_at">
          <span>·</span>
          <span>最后处理 <KunTime :time="item.last_event_at" /></span>
        </template>
      </div>

      <slot name="note" />
    </div>

    <div class="flex shrink-0 gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
