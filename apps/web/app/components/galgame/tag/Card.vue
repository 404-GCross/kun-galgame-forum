<script setup lang="ts">
import { KUN_GALGAME_TAG_CATEGORY_MAP } from '~/constants/galgameTag'

const props = defineProps<{
  tag: GalgameTagItem | GalgameTaxonomySearchItem
}>()

const category = computed(() =>
  'category' in props.tag ? props.tag.category : undefined
)
const count = computed(() =>
  'galgame_count' in props.tag ? props.tag.galgame_count : undefined
)
const tooltip = computed(() =>
  category.value === undefined || count.value === undefined
    ? props.tag.name
    : `${KUN_GALGAME_TAG_CATEGORY_MAP[category.value]} - 含有 ${count.value} 个 Galgame`
)
</script>

<template>
  <KunTooltip :text="tooltip">
    <KunCard
      :is-transparent="false"
      :is-hoverable="true"
      :href="`/galgame/tag/${tag.id}`"
    >
      <h3 class="text-default-900 font-semibold">
        <span
          :class="
            cn(
              'mr-1.5',
              category === 'content' && 'text-primary',
              category === 'sexual' && 'text-danger',
              category === 'technical' && 'text-success'
            )
          "
        >
          #
        </span>
        {{ tag.name }}
        <KunChip v-if="count !== undefined" size="xs">
          {{ `+ ${count}` }}
        </KunChip>
      </h3>
    </KunCard>
  </KunTooltip>
</template>
