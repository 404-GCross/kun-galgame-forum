<script setup lang="ts">
// One 会社 box on the graph. A real <button>, not a styled div: the graph is
// navigated with the keyboard as much as with the mouse, and everything that
// makes that work — focus ring, Enter/Space, being announced as actionable —
// comes free from the element and has to be rebuilt by hand otherwise.
//
// Sizing is fixed rather than content-driven (the constants the layout placed
// it by), because the layout computed its coordinates on the server without a
// DOM to measure: a box that grew to fit a long brand name would overlap the
// neighbour the layout carefully spaced it from.
const props = defineProps<{
  node: GalgameOfficialGraphPlacedNode
  isCurrent: boolean
  /** On the highlighted path: the active node itself or one of its neighbours. */
  isActive: boolean
  /** Something else is highlighted, so this one steps back. */
  isDimmed: boolean
  isSelected: boolean
  /** Roving tabindex — exactly one node in the graph is in the tab order. */
  isTabStop: boolean
}>()

// The `_mini` (360px) variant: this is a 28px thumbnail and the original is
// pure waste. '' (no logo) renders no frame rather than an empty box.
const logoSrc = computed(() =>
  props.node.official.logo
    ? withImageVariant(props.node.official.logo, 'mini')
    : ''
)

const label = computed(() => {
  const { name, work_count } = props.node.official
  const held = work_count > 0 ? `, 收录 ${work_count} 部作品` : ''
  return props.isCurrent
    ? `${name}, 当前会社${held}`
    : `${name}${held}, 按 Enter 前往`
})
</script>

<template>
  <button
    type="button"
    :style="{
      left: `${node.x - OFFICIAL_GRAPH_NODE_WIDTH / 2}px`,
      top: `${node.y - OFFICIAL_GRAPH_NODE_HEIGHT / 2}px`,
      width: `${OFFICIAL_GRAPH_NODE_WIDTH}px`,
      height: `${OFFICIAL_GRAPH_NODE_HEIGHT}px`
    }"
    :tabindex="isTabStop ? 0 : -1"
    :aria-current="isCurrent ? 'page' : undefined"
    :aria-label="label"
    :class="
      cn(
        'absolute flex items-center gap-2 rounded-xl border px-2.5 text-left',
        'focus-visible:ring-primary-500 outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
        'transition-[opacity,box-shadow,border-color] duration-200',
        isCurrent
          ? 'border-primary-500 bg-primary-50'
          : 'border-default-200 bg-background hover:border-default-400',
        isSelected && !isCurrent && 'border-primary-400',
        isActive && 'shadow-md',
        isDimmed && 'opacity-35'
      )
    "
  >
    <!-- object-contain on its own light surface, never cover: a brand mark is
         usually a transparent PNG in one dark colour, and cropping it to a
         square makes it a different logo. -->
    <div v-if="logoSrc" class="bg-default-100 shrink-0 rounded-md p-0.5">
      <KunImage
        :src="logoSrc"
        :alt="`${node.official.name} logo`"
        loading="lazy"
        object-fit="contain"
        class-name="size-7"
      />
    </div>

    <span class="min-w-0 flex-1">
      <span
        :class="
          cn(
            'block truncate text-sm font-medium',
            isCurrent ? 'text-primary-600' : 'text-foreground'
          )
        "
      >
        {{ node.official.name }}
      </span>
      <!-- Only when there is something to count: a "0 部" line under every
           brand a publisher ever registered is noise. -->
      <span
        v-if="node.official.work_count > 0"
        class="text-default-400 block text-xs"
      >
        {{ `${node.official.work_count} 部` }}
      </span>
    </span>

    <KunIcon
      v-if="isSelected"
      name="lucide:chevron-right"
      class-name="text-primary-500 shrink-0"
    />
  </button>
</template>
