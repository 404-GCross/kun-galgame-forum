<script setup lang="ts">
const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as ReplyActivityData | undefined
)
const quoted = computed(() => data.value?.quoted_reply)
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-2">
      <ActivityCardQuote
        v-if="quoted"
        :content="quoted.content"
        :label="`#${quoted.floor}`"
      />

      <KunContent
        compact
        class="text-base"
        :content="renderKatex(activity.content)"
      />

      <KunLink
        v-if="data?.topic_title"
        underline="none"
        color="default"
        :to="replyPermalink(activity.link, data?.floor)"
        class-name="text-default-500 hover:text-primary flex items-center gap-1 text-sm"
      >
        <KunIcon name="icon-park-outline:topic" class="size-4 shrink-0" />
        <span class="line-clamp-1">{{ data.topic_title }}</span>
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
