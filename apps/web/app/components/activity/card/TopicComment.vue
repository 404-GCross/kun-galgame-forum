<script setup lang="ts">
const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as TopicCommentActivityData | undefined
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

      <ActivityCollapse :max-height="300">
        <p class="text-default-700 text-base break-all whitespace-pre-line">
          {{ markdownToText(activity.content, { preserveNewlines: true }) }}
        </p>
      </ActivityCollapse>

      <KunLink
        v-if="data?.topic_title"
        underline="none"
        color="default"
        :to="commentPermalink(activity.link, data?.comment_id)"
        class-name="text-default-500 hover:text-primary flex items-center gap-1 text-sm"
      >
        <KunIcon name="icon-park-outline:topic" class="size-4 shrink-0" />
        <span class="line-clamp-1">{{ data.topic_title }}</span>
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
