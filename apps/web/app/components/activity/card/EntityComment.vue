<script setup lang="ts">
const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as EntityRefActivityData | undefined
)

const meta = computed(() =>
  props.activity.type === 'TOOLSET_COMMENT_CREATION'
    ? { icon: 'lucide:wrench', label: '工具' }
    : { icon: 'lucide:globe', label: '网站' }
)
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-1.5">
      <p class="text-default-700 line-clamp-4 text-base break-all whitespace-pre-line">
        {{ markdownToText(activity.content) }}
      </p>
      <KunLink
        underline="none"
        color="default"
        :to="activity.link"
        class-name="text-default-500 hover:text-primary inline-flex items-center gap-1 text-sm"
      >
        <KunIcon :name="meta.icon" class="size-3.5 shrink-0" />
        {{ meta.label }}《{{ data?.parent_name }}》
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
