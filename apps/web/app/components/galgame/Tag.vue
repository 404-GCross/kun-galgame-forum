<script setup lang="ts">
import {
  KUN_GALGAME_TAG_CATEGORY_MAP,
  KUN_GALGAME_TAG_SPOILER_MAP,
  type KunGalgameTagCategory,
  type KunGalgameTagSpoiler
} from '~/constants/galgameTag'

const props = withDefaults(
  defineProps<{
    tags: GalgameDetailTag[]
    // 'mobile' renders the original inline-checkbox filter + larger chips and is
    // mounted right under the page header; 'desktop' keeps the compact popover
    // filter + small chips in the sidebar. Two instances are mounted (one per
    // breakpoint via Galgame.vue) because position, chip size and filter UI all
    // differ between them.
    variant?: 'mobile' | 'desktop'
  }>(),
  { variant: 'desktop' }
)

const isMobile = computed(() => props.variant === 'mobile')

// Adult tags are off by default and only pre-selected when the viewer has NSFW
// enabled (cookie-persisted, so this is correct at SSR setup — same source the
// page's isNsfwMode reads in [gid]/index.vue). A SFW viewer can still opt in by
// ticking the 成人内容 checkbox.
const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
const isNsfwEnabled = computed(
  () =>
    showKUNGalgameContentLimit.value === 'nsfw' ||
    showKUNGalgameContentLimit.value === 'all'
)

// Default filter: 游戏内容 + 技术细节 (+ 成人内容 when NSFW is on), 无剧透 only.
const selectedCategories = ref<KunGalgameTagCategory[]>(
  isNsfwEnabled.value
    ? ['content', 'technical', 'sexual']
    : ['content', 'technical']
)
const selectedSpoilerLevels = ref<KunGalgameTagSpoiler[]>([0])

const toggleItemInArray = <T,>(arrayRef: Ref<T[]>, item: T) => {
  const index = arrayRef.value.indexOf(item)
  if (index === -1) {
    arrayRef.value.push(item)
  } else {
    arrayRef.value.splice(index, 1)
  }
}

const toggleCategory = (category: KunGalgameTagCategory) => {
  toggleItemInArray(selectedCategories, category)
}

const toggleSpoilerLevel = (spoiler: KunGalgameTagSpoiler) => {
  toggleItemInArray(selectedSpoilerLevels, spoiler)
}

const filteredTags = computed(() => {
  if (
    selectedCategories.value.length === 0 ||
    selectedSpoilerLevels.value.length === 0
  ) {
    return []
  }

  const filtered = props.tags.filter(
    (tag) =>
      selectedCategories.value.includes(tag.category) &&
      selectedSpoilerLevels.value.includes(tag.spoiler_level as 0)
  )
  return filtered.sort((a, b) => a.id - b.id)
})

// Color of the trailing "+N" badge encodes the tag's category. Same
// mapping the leading "#" used to carry before the redesign:
//   content   → primary (blue)
//   sexual    → danger  (red)
//   technical → success (green)
const countColorByCategory = (category: string): string => {
  if (category === 'content') return 'text-primary'
  if (category === 'sexual') return 'text-danger'
  if (category === 'technical') return 'text-success'
  return 'text-default-500'
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    class-name="overflow-visible"
    content-class="space-y-3"
  >
    <!-- Mobile: the original inline-checkbox filter (horizontal scroll), above
         the tag list. The sidebar/desktop variant uses the popover below. -->
    <KunScrollShadow v-if="isMobile" shadow-size="5rem">
      <div class="flex w-fit items-center gap-3 whitespace-nowrap">
        <KunCheckBox
          v-for="(name, key) in KUN_GALGAME_TAG_CATEGORY_MAP"
          :key="key"
          class-name="gap-2"
          :model-value="selectedCategories.includes(key)"
          color="primary"
          @click="toggleCategory(key)"
        >
          {{ name }}
        </KunCheckBox>

        <KunCheckBox
          v-for="(name, key) in KUN_GALGAME_TAG_SPOILER_MAP"
          :key="key"
          class-name="gap-2"
          :model-value="selectedSpoilerLevels.includes(Number(key) as 0)"
          color="primary"
          @click="toggleSpoilerLevel(Number(key) as KunGalgameTagSpoiler)"
        >
          {{ name }}
        </KunCheckBox>
      </div>
    </KunScrollShadow>

    <KunScrollShadow
      axis="vertical"
      shadow-size="3rem"
      :class-name="isMobile ? 'max-h-[300px]' : 'max-h-[200px] md:max-h-[400px]'"
    >
      <TransitionGroup name="tag-list" tag="div" class="flex flex-wrap gap-1.5">
        <KunLink
          v-for="tag in filteredTags"
          :key="tag.id"
          underline="none"
          :to="`/galgame-tag/${tag.id}`"
        >
          <KunChip
            class-name="bg-default-500/10 cursor-pointer"
            :size="isMobile ? 'md' : 'sm'"
          >
            {{ tag.name }}
            <span :class="cn('text-xs', countColorByCategory(tag.category))">
              {{ `+${tag.galgame_count}` }}
            </span>
            <span v-if="tag.spoiler_level > 0" class="text-warning-600 text-xs">
              {{ tag.spoiler_level > 1 ? '(严重剧透)' : '(剧透)' }}
            </span>
          </KunChip>
        </KunLink>
      </TransitionGroup>

      <KunNull
        v-if="filteredTags.length === 0"
        description="请至少选择一个类别来查看标签，或调整剧透等级"
      />
    </KunScrollShadow>

    <!-- Desktop: compact popover filter (fits the narrow sidebar column). -->
    <KunPopover v-if="!isMobile" position="top-start">
      <template #trigger>
        <KunButton variant="flat" color="primary" size="sm" full-width>
          <KunIcon name="lucide:filter" />
          筛选标签
        </KunButton>
      </template>

      <div class="min-w-[240px] space-y-4 p-4">
        <div class="space-y-2">
          <p class="text-default-500 text-xs font-medium">标签类型</p>
          <div class="flex flex-wrap gap-3">
            <KunCheckBox
              v-for="(name, key) in KUN_GALGAME_TAG_CATEGORY_MAP"
              :key="key"
              class-name="gap-2"
              :model-value="selectedCategories.includes(key)"
              color="primary"
              @click="toggleCategory(key)"
            >
              {{ name }}
            </KunCheckBox>
          </div>
        </div>

        <div class="space-y-2">
          <p class="text-default-500 text-xs font-medium">剧透等级</p>
          <div class="flex flex-wrap gap-3">
            <KunCheckBox
              v-for="(name, key) in KUN_GALGAME_TAG_SPOILER_MAP"
              :key="key"
              class-name="gap-2"
              :model-value="selectedSpoilerLevels.includes(Number(key) as 0)"
              color="primary"
              @click="toggleSpoilerLevel(Number(key) as KunGalgameTagSpoiler)"
            >
              {{ name }}
            </KunCheckBox>
          </div>
        </div>
      </div>
    </KunPopover>
  </KunCard>
</template>

<style scoped>
.tag-list-move,
.tag-list-enter-active,
.tag-list-leave-active {
  transition: all 0.5s cubic-bezier(0.55, 0, 0.1, 1);
}
.tag-list-enter-from,
.tag-list-leave-to {
  opacity: 0;
  transform: scale(0.8);
}
.tag-list-leave-active {
  position: absolute;
}
</style>
