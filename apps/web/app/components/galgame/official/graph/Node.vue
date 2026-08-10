<script setup lang="ts">
const props = defineProps<{
  node: GalgameOfficialGraphPlacedNode
  isCurrent: boolean
  isActive: boolean
  isDimmed: boolean
  isSelected: boolean
}>()

const logoSrc = computed(() =>
  props.node.official.logo
    ? withImageVariant(props.node.official.logo, 'mini')
    : ''
)

const label = computed(() => {
  const { name, work_count } = props.node.official
  const held = work_count > 0 ? `, 收录 ${work_count} 部作品` : ''
  return props.isCurrent ? `${name}, 当前会社${held}` : `${name}${held}`
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
    tabindex="-1"
    :aria-current="isCurrent ? 'page' : undefined"
    :aria-label="label"
    :class="
      cn(
        'absolute flex items-center gap-2 rounded-xl border px-2.5 text-left outline-none',
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
