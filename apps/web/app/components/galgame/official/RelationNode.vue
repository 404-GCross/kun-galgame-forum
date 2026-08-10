<script setup lang="ts">
import { KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP } from '~/constants/galgameOfficial'

const props = defineProps<{
  official: GalgameOfficialRelationNode
  role?: string
  isCurrent?: boolean
}>()

const logoSrc = computed(() =>
  props.official.logo ? withImageVariant(props.official.logo, 'mini') : ''
)

const roleText = computed(() =>
  props.role ? (KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP[props.role] ?? '') : ''
)
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="!isCurrent"
    :href="isCurrent ? undefined : taxonomyDetailPath('official', official.id)"
    padding="sm"
    :class-name="cn('w-full', isCurrent && 'border-primary-500 bg-primary-50')"
    content-class="flex-row items-center gap-2 sm:gap-3"
  >
    <div v-if="logoSrc" class="bg-default-100 shrink-0 rounded-md p-1">
      <KunImage
        :src="logoSrc"
        :alt="`${official.name} logo`"
        loading="lazy"
        object-fit="contain"
        class-name="size-8 sm:size-10"
      />
    </div>

    <span
      :class="
        cn(
          'min-w-0 truncate font-medium',
          isCurrent ? 'text-primary-600' : 'text-foreground'
        )
      "
    >
      {{ official.name }}
    </span>

    <KunChip
      v-if="roleText"
      size="xs"
      :color="isCurrent ? 'primary' : 'default'"
      class-name="shrink-0"
    >
      {{ roleText }}
    </KunChip>

    <KunChip
      v-if="official.work_count > 0"
      size="xs"
      color="success"
      class-name="ml-auto shrink-0"
    >
      {{ `${official.work_count} 部` }}
    </KunChip>
  </KunCard>
</template>
