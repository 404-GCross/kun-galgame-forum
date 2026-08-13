<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { KUN_GALGAME_OFFICIAL_GRAPH_EDGE_MAP } from '~/constants/galgameOfficial'

const props = defineProps<{
  layout: GalgameOfficialGraphLayout
  currentId: number
  isFullscreen?: boolean
}>()

const emit = defineEmits<{
  open: [id: number]
  expand: []
}>()

const selectedId = defineModel<number | null>('selectedId', { required: true })

const viewport = ref<HTMLElement | null>(null)
const content = computed(() => ({
  width: props.layout.width,
  height: props.layout.height
}))

const focus = computed(() => {
  const node = props.layout.nodes.find((n) => n.official.id === props.currentId)
  return node ? { x: node.x, y: node.y } : null
})

const isWide = useMediaQuery('(min-width: 640px)')
const isInteractive = computed(() => props.isFullscreen || isWide.value)
const isPreview = computed(() => !isInteractive.value)

const freeTouchPan = computed(() => !!props.isFullscreen)

const {
  scale,
  offsetX,
  offsetY,
  isPanning,
  transform,
  fit,
  zoomBy,
  centerOn,
  onPointerDown,
  onPointerMove,
  onPointerUp,
  onWheel
} = useOfficialGraphViewport(viewport, content, {
  focus,
  freeTouchPan,
  preview: isPreview
})

const EXPAND_AT = 1.15
let hasOfferedFullscreen = false
watch(scale, (value) => {
  if (props.isFullscreen || hasOfferedFullscreen || value < EXPAND_AT) return
  hasOfferedFullscreen = true
  emit('expand')
})

const hoveredId = ref<number | null>(null)

const hasPointed = ref(false)
watch([selectedId, hoveredId], () => {
  hasPointed.value = true
})

const activeId = computed(() => {
  if (isPreview.value) return props.currentId
  if (hoveredId.value !== null) return hoveredId.value
  if (!hasPointed.value) return null
  return selectedId.value
})

const neighbours = computed(() => {
  const map = new Map<number, Set<number>>()
  for (const edge of props.layout.edges) {
    if (!map.has(edge.from)) map.set(edge.from, new Set())
    if (!map.has(edge.to)) map.set(edge.to, new Set())
    map.get(edge.from)!.add(edge.to)
    map.get(edge.to)!.add(edge.from)
  }
  return map
})

const isLit = (id: number) =>
  isPreview.value ||
  activeId.value === null ||
  activeId.value === id ||
  (neighbours.value.get(activeId.value)?.has(id) ?? false)

const isEdgeLit = (edge: GalgameOfficialGraphEdge) =>
  activeId.value !== null &&
  (edge.from === activeId.value || edge.to === activeId.value)

const activate = (id: number) => {
  if (selectedId.value === id && id !== props.currentId) emit('open', id)
  else selectedId.value = id
}

const homeToCurrent = () => {
  const node = props.layout.nodes.find((n) => n.official.id === props.currentId)
  if (node) centerOn(node.x, node.y)
}

const edgeLabel = (edge: GalgameOfficialGraphEdge) =>
  KUN_GALGAME_OFFICIAL_GRAPH_EDGE_MAP[edge.kind] ?? ''

const edgeTone = (edge: GalgameOfficialGraphEdge) => {
  if (isEdgeLit(edge)) return 'primary-500'
  if (edge.kind === 'succession') return 'secondary-300'
  if (edge.kind === 'spawn') return 'warning-300'
  return 'default-300'
}
const edgeStroke = (edge: GalgameOfficialGraphEdge) =>
  ({
    'primary-500': 'stroke-primary-500',
    'secondary-300': 'stroke-secondary-300',
    'warning-300': 'stroke-warning-300',
    'default-300': 'stroke-default-300'
  })[edgeTone(edge)]
const edgeFill = (edge: GalgameOfficialGraphEdge) =>
  ({
    'primary-500': 'fill-primary-500',
    'secondary-300': 'fill-secondary-300',
    'warning-300': 'fill-warning-300',
    'default-300': 'fill-default-300'
  })[edgeTone(edge)]

const DRAG_SLOP = 4
let pressedAt = { x: 0, y: 0 }

const onSurfacePointerDown = (event: PointerEvent) => {
  if (isPreview.value) return
  pressedAt = { x: event.clientX, y: event.clientY }
  onPointerDown(event)
}

const onSurfaceClick = (event: MouseEvent) => {
  if (isPreview.value) {
    emit('expand')
    return
  }
  const moved = Math.hypot(
    event.clientX - pressedAt.x,
    event.clientY - pressedAt.y
  )
  if (moved <= DRAG_SLOP) selectedId.value = null
}

const onSurfaceWheel = (event: WheelEvent) => {
  if (isPreview.value) return
  onWheel(event)
}

const uid = useId()
const svgId = (name: string) => `kun-official-graph-${name}-${uid}`
</script>

<template>
  <div
    ref="viewport"
    :aria-label="
      isPreview
        ? '会社关系图预览, 点击全屏查看'
        : '会社关系图, 拖拽平移, 滚轮缩放'
    "
    :class="
      cn(
        'border-default-200 bg-default-50 relative overflow-hidden rounded-xl border select-none',
        // The preview is a thumbnail and should look like one: a band, not a
        // canvas with nothing in it.
        isFullscreen ? 'h-full' : isPreview ? 'h-48' : 'h-[26rem] sm:h-[32rem]',
        // Fullscreen owns every gesture; the inline canvas leaves the page its
        // vertical scroll; the preview takes nothing at all.
        isFullscreen ? 'touch-none' : isPreview ? 'touch-auto' : 'touch-pan-y',
        !isInteractive && 'cursor-zoom-in',
        isInteractive && (isPanning ? 'cursor-grabbing' : 'cursor-grab')
      )
    "
    @pointerdown="onSurfacePointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
    @click.self="onSurfaceClick"
    @wheel="onSurfaceWheel"
  >
    <svg
      v-if="!isPreview"
      class="pointer-events-none absolute inset-0 size-full"
      aria-hidden="true"
    >
      <defs>
        <pattern
          :id="svgId('grid')"
          width="26"
          height="26"
          patternUnits="userSpaceOnUse"
          :patternTransform="`translate(${offsetX} ${offsetY}) scale(${scale})`"
        >
          <path
            d="M 26 0 L 0 0 0 26"
            fill="none"
            stroke-width="1"
            class="stroke-default-200"
          />
        </pattern>
      </defs>
      <rect width="100%" height="100%" :fill="`url(#${svgId('grid')})`" />
    </svg>

    <div
      :class="
        cn(
          'absolute top-0 left-0 origin-top-left',
          isPreview && 'pointer-events-none'
        )
      "
      :style="{ transform }"
    >
      <svg
        :width="layout.width"
        :height="layout.height"
        :viewBox="`0 0 ${layout.width} ${layout.height}`"
        class="pointer-events-none absolute top-0 left-0 overflow-visible"
        aria-hidden="true"
      >
        <path
          v-for="edge in layout.edges"
          :key="`line-${edge.id}`"
          :d="edge.path"
          fill="none"
          stroke-linecap="round"
          stroke-linejoin="round"
          :stroke-width="isEdgeLit(edge) ? 2.25 : 1.5"
          :stroke-dasharray="
            edge.kind === 'succession' || edge.kind === 'spawn'
              ? '5 4'
              : undefined
          "
          :class="
            cn(
              'transition-[stroke,stroke-width,opacity] duration-200',
              edgeStroke(edge),
              activeId !== null && !isEdgeLit(edge) && 'opacity-20'
            )
          "
        />

        <path
          v-for="edge in layout.edges"
          :key="`head-${edge.id}`"
          d="M -9 -3.6 L 0 0 L -9 3.6 L -6.4 0 Z"
          :transform="`translate(${edge.head.x} ${edge.head.y}) rotate(${edge.head.angle})`"
          :class="
            cn(
              'transition-opacity duration-200',
              edgeFill(edge),
              activeId !== null && !isEdgeLit(edge) && 'opacity-20'
            )
          "
        />

        <text
          v-for="edge in layout.edges.filter(isEdgeLit)"
          :key="`label-${edge.id}`"
          :x="edge.labelX"
          :y="edge.labelY"
          text-anchor="middle"
          dominant-baseline="middle"
          stroke-width="4"
          paint-order="stroke"
          class="fill-primary-600 stroke-default-50 text-[11px] font-medium"
        >
          {{ edgeLabel(edge) }}
        </text>
      </svg>

      <GalgameOfficialGraphNode
        v-for="node in layout.nodes"
        :key="node.official.id"
        :data-node-id="node.official.id"
        :node="node"
        :is-current="node.official.id === currentId"
        :is-active="activeId === node.official.id"
        :is-dimmed="!isLit(node.official.id)"
        :is-selected="selectedId === node.official.id"
        @pointerdown.stop
        @mouseenter="hoveredId = node.official.id"
        @mouseleave="hoveredId = null"
        @click="activate(node.official.id)"
        @dblclick="
          node.official.id !== currentId && emit('open', node.official.id)
        "
      />
    </div>

    <div
      v-if="isInteractive"
      class="absolute top-2 right-2 flex flex-col gap-1"
      @pointerdown.stop
      @click.stop
    >
      <KunButton
        :is-icon-only="true"
        variant="flat"
        color="default"
        size="sm"
        aria-label="放大"
        @click="zoomBy(1.2)"
      >
        <KunIcon name="lucide:plus" />
      </KunButton>
      <KunButton
        :is-icon-only="true"
        variant="flat"
        color="default"
        size="sm"
        aria-label="缩小"
        @click="zoomBy(1 / 1.2)"
      >
        <KunIcon name="lucide:minus" />
      </KunButton>
      <KunButton
        :is-icon-only="true"
        variant="flat"
        color="default"
        size="sm"
        aria-label="适应画布"
        @click="fit()"
      >
        <KunIcon name="lucide:scan" />
      </KunButton>
      <KunButton
        :is-icon-only="true"
        variant="flat"
        color="primary"
        size="sm"
        aria-label="回到本会社"
        @click="homeToCurrent"
      >
        <KunIcon name="lucide:crosshair" />
      </KunButton>
    </div>

    <div
      class="absolute right-3 bottom-2 left-3 flex items-center gap-2"
      @pointerdown.stop
    >
      <p v-if="isWide" class="text-default-400 pointer-events-none text-xs">
        滚轮缩放 · 拖拽平移 · 点击选中 · 双击打开
      </p>
      <p
        v-else-if="isFullscreen"
        class="text-default-400 pointer-events-none text-xs"
      >
        双指缩放 · 拖动查看 · 点按选中
      </p>

      <KunButton
        v-if="!isFullscreen"
        variant="flat"
        color="primary"
        size="sm"
        :full-width="isPreview"
        :class-name="isPreview ? undefined : 'ml-auto shrink-0'"
        @click.stop="emit('expand')"
      >
        <KunIcon name="lucide:maximize" />
        {{ isPreview ? '查看完整关系图' : '全屏查看' }}
      </KunButton>
    </div>
  </div>
</template>
