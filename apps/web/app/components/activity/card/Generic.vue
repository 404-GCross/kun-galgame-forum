<script setup lang="ts">
defineProps<{ activity: ActivityItem }>()
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <KunLink
      underline="none"
      color="default"
      :to="activity.link"
      class-name="hover:text-primary block break-all transition-colors"
    >
      <KunText
        class-name="whitespace-normal!"
        :content="markdownToText(activity.content)"
      />
    </KunLink>
  </ActivityCardShell>
</template>
