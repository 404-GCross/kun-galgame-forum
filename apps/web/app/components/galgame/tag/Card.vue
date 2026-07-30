<script setup lang="ts">
import { KUN_GALGAME_TAG_CATEGORY_MAP } from '~/constants/galgameTag'

defineProps<{
  tag: GalgameTagItem
}>()
</script>

<template>
  <KunTooltip
    :text="`${KUN_GALGAME_TAG_CATEGORY_MAP[tag.category]} - 含有 ${tag.galgame_count} 个 Galgame`"
  >
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
              tag.category === 'content' && 'text-primary',
              tag.category === 'sexual' && 'text-danger',
              tag.category === 'technical' && 'text-success'
            )
          "
        >
          #
        </span>
        {{ tag.name }}
        <KunChip size="xs">
          {{ `+ ${tag.galgame_count}` }}
        </KunChip>
      </h3>
    </KunCard>
  </KunTooltip>
</template>
