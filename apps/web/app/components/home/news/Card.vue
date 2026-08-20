<script setup lang="ts">
const props = defineProps<{
  item: KunNewsItem
  source: KunNewsSource | undefined
}>()

const sourceName = computed(() => props.source?.name ?? '合作站点')
</script>

<template>
  <KunCard is-hoverable padding="md" content-class="gap-0">
    <div class="flex flex-col gap-3 sm:flex-row sm:gap-4">
      <KunImage
        v-if="item.banner_url"
        :src="item.banner_url"
        :alt="item.title"
        aspect-ratio="16/9"
        object-fit="cover"
        loading="lazy"
        class-name="w-full shrink-0 overflow-hidden rounded-lg sm:w-48"
      />

      <div class="flex min-w-0 flex-1 flex-col gap-1.5">
        <div class="flex items-start gap-2">
          <KunChip
            v-if="item.lane === 'column'"
            size="sm"
            color="secondary"
            variant="flat"
            class="mt-0.5 shrink-0"
          >
            专栏
          </KunChip>
          <KunLink
            :href="item.source_url"
            target="_blank"
            color="default"
            underline="none"
            class-name="hover:text-primary line-clamp-2 text-base font-medium transition-colors"
          >
            {{ item.title }}
          </KunLink>
        </div>

        <p class="text-default-500 line-clamp-2 text-sm">
          {{ item.preview }}
        </p>

        <div
          class="text-default-400 mt-auto flex flex-wrap items-center gap-x-2 gap-y-1 pt-1 text-xs"
        >
          <span>{{ formatTimeDifference(item.published_at) }}</span>
          <span aria-hidden="true">·</span>
          <span class="truncate">{{ sourceName }}</span>
          <KunLink
            :href="item.source_url"
            target="_blank"
            color="primary"
            size="sm"
            underline="hover"
            is-show-anchor-icon
            class-name="ml-auto shrink-0"
          >
            阅读原文
          </KunLink>
        </div>
      </div>
    </div>
  </KunCard>
</template>
