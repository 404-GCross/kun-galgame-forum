<script setup lang="ts">
import { KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP } from '~/constants/galgameOfficial'

// One 会社 row inside the family tree: brand mark, name, what it is to the row
// above, and how much of the catalogue it holds.
//
// The 会社 whose page this is renders as a highlighted, NON-clickable row — a
// link back to the page you are already on is the one navigation a tree must
// not offer, and the highlight is what tells the reader where they are in a
// family of a dozen brands.
const props = defineProps<{
  official: GalgameOfficialRelationNode
  /** A key of KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP; omitted renders no chip. */
  role?: string
  isCurrent?: boolean
}>()

// The `_mini` (360px) variant: these are 32–40px thumbnails, the original is
// pure waste. '' (no logo) renders no frame at all rather than an empty box.
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
    <!-- object-contain on its own light surface, never cover: a brand mark is
         usually a transparent PNG in one dark colour, and cropping it to a
         square makes it a different logo. -->
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

    <!-- Only when there is something to count: a "0 部" badge on every brand a
         publisher ever registered is noise the tree does not need. -->
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
