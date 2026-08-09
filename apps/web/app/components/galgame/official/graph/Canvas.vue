<script setup lang="ts">
// The drawing surface: edges in one SVG, nodes as absolutely positioned
// buttons above it, both riding a single CSS transform so pan and zoom cost
// one composited property instead of a relayout per frame.
//
// Nodes are HTML rather than SVG <foreignObject> deliberately — text rendering,
// truncation, focus rings and image loading all behave normally in HTML and all
// have sharp edges inside SVG.
//
// Interaction model, which is the part worth stating: hovering PREVIEWS a node
// (its edges light, everything else steps back), clicking SELECTS it (the panel
// beside the graph fills in), and acting on an already-selected node OPENS it.
// One gesture per level of commitment, and the same three levels on the
// keyboard — arrows preview, Enter selects, Enter again opens — so nobody
// navigates away from the page by brushing past a box.
import { KUN_GALGAME_OFFICIAL_GRAPH_EDGE_MAP } from '~/constants/galgameOfficial'

const props = defineProps<{
  layout: GalgameOfficialGraphLayout
  currentId: number
  /** Set by the fullscreen host: fills its box, and stops offering to go
   * fullscreen from inside fullscreen. */
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

// Where the opening view looks when the whole family will not fit legibly.
const focus = computed(() => {
  const node = props.layout.nodes.find((n) => n.official.id === props.currentId)
  return node ? { x: node.x, y: node.y } : null
})

const {
  scale,
  fitScale,
  offsetX,
  offsetY,
  isPanning,
  transform,
  fit,
  zoomBy,
  centerOn,
  reveal,
  onPointerDown,
  onPointerMove,
  onPointerUp,
  onWheel
} = useOfficialGraphViewport(viewport, content, focus)

/**
 * Zooming in inside a 26rem strip is the reader saying the box is too small.
 * Taking them fullscreen at that point is answering the request rather than
 * making them find a button — but only ONCE per visit: a modal that reappears
 * every time you lean on the wheel is a modal you start fighting.
 */
const EXPAND_AT = 1.15
let hasOfferedFullscreen = false
watch(scale, (value) => {
  if (props.isFullscreen || hasOfferedFullscreen || value < EXPAND_AT) return
  hasOfferedFullscreen = true
  emit('expand')
})

const hoveredId = ref<number | null>(null)
const focusedId = ref<number | null>(null)

// What the highlight follows, most immediate intent first.
const activeId = computed(
  () => hoveredId.value ?? focusedId.value ?? selectedId.value
)

// A node's neighbourhood, precomputed once per layout rather than scanned per
// node per render: 60 nodes × 60 edges is small, but this runs on every hover.
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
  activeId.value === null ||
  activeId.value === id ||
  (neighbours.value.get(activeId.value)?.has(id) ?? false)

const isEdgeLit = (edge: GalgameOfficialGraphEdge) =>
  activeId.value !== null &&
  (edge.from === activeId.value || edge.to === activeId.value)

const focusNode = (id: number) => {
  focusedId.value = id
  const node = props.layout.nodes.find((n) => n.official.id === id)
  if (node) {
    reveal(
      node.x,
      node.y,
      OFFICIAL_GRAPH_NODE_WIDTH,
      OFFICIAL_GRAPH_NODE_HEIGHT
    )
  }
  nextTick(() => {
    viewport.value
      ?.querySelector<HTMLElement>(`[data-node-id="${id}"]`)
      ?.focus({ preventScroll: true })
  })
}

/** Select, then open: the second activation of the same node is the one that
 * leaves the page. The 会社 whose page this is can never be opened — a link
 * back to where you already are is the one navigation a graph must not offer. */
const activate = (id: number) => {
  if (selectedId.value === id && id !== props.currentId) emit('open', id)
  else selectedId.value = id
}

/** Arrow keys move to the nearest node in that direction — geometric, not
 * index-based, because "the box above this one" is what the reader means and
 * the layout order has no idea which that is. Perpendicular distance is
 * weighted heavier so pressing ↑ prefers the parent directly overhead to a
 * cousin further along the same row. */
const step = (dx: number, dy: number) => {
  const from = props.layout.nodes.find(
    (n) =>
      n.official.id === (focusedId.value ?? selectedId.value ?? props.currentId)
  )
  if (!from) return

  let best: { id: number; cost: number } | null = null
  for (const node of props.layout.nodes) {
    if (node.official.id === from.official.id) continue
    const along = (node.x - from.x) * dx + (node.y - from.y) * dy
    const across = Math.abs((node.x - from.x) * dy + (node.y - from.y) * dx)
    if (along <= 1) continue
    const cost = along + across * 2
    if (!best || cost < best.cost) best = { id: node.official.id, cost }
  }
  if (best) focusNode(best.id)
}

const homeToCurrent = () => {
  const node = props.layout.nodes.find((n) => n.official.id === props.currentId)
  if (!node) return
  centerOn(node.x, node.y)
  focusNode(node.official.id)
}

const onKeydown = (event: KeyboardEvent) => {
  // Escape peels one layer at a time: the selection first, and only once there
  // is nothing selected does it reach the fullscreen modal behind it. Swallowing
  // it unconditionally would make the graph the one thing on the site you
  // cannot Escape out of.
  if (event.key === 'Escape') {
    if (selectedId.value === null) return
    event.preventDefault()
    selectedId.value = null
    return
  }

  const handlers: Record<string, () => void> = {
    ArrowUp: () => step(0, -1),
    ArrowDown: () => step(0, 1),
    ArrowLeft: () => step(-1, 0),
    ArrowRight: () => step(1, 0),
    Home: homeToCurrent,
    c: homeToCurrent,
    f: fit,
    '+': () => zoomBy(1.2),
    '=': () => zoomBy(1.2),
    '-': () => zoomBy(1 / 1.2),
    '0': fit
  }
  const handler = handlers[event.key]
  if (!handler) return
  event.preventDefault()
  handler()
}

// The reader's own entry point into the graph: whichever node is live, or the
// 会社 they came for. Exactly one node is ever in the tab order — sixty tab
// stops in the middle of a page is not navigation, it is a wall.
const tabStopId = computed(
  () => focusedId.value ?? selectedId.value ?? props.currentId
)

const edgeLabel = (edge: GalgameOfficialGraphEdge) =>
  KUN_GALGAME_OFFICIAL_GRAPH_EDGE_MAP[edge.kind] ?? ''

// Clicking the empty canvas clears the selection — but only when it was a
// click and not the end of a drag, which is the difference between dismissing
// the panel and losing it every time you pan.
const DRAG_SLOP = 4
let pressedAt = { x: 0, y: 0 }

const onSurfacePointerDown = (event: PointerEvent) => {
  pressedAt = { x: event.clientX, y: event.clientY }
  onPointerDown(event)
}

const onSurfaceClick = (event: MouseEvent) => {
  const moved = Math.hypot(
    event.clientX - pressedAt.x,
    event.clientY - pressedAt.y
  )
  if (moved <= DRAG_SLOP) selectedId.value = null
}

// Marker ids must be unique per instance: two graphs on one page sharing an id
// would both resolve to whichever <defs> the document happened to parse first.
const uid = useId()
const svgId = (name: string) => `kun-official-graph-${name}-${uid}`
</script>

<template>
  <div
    ref="viewport"
    role="application"
    aria-label="会社关系图, 方向键移动, 回车选中, 再次回车打开"
    :class="
      cn(
        'border-default-200 bg-default-50 relative touch-pan-y overflow-hidden rounded-xl border select-none',
        isFullscreen ? 'h-full' : 'h-[26rem] sm:h-[32rem]',
        isPanning ? 'cursor-grabbing' : 'cursor-grab'
      )
    "
    @pointerdown="onSurfacePointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
    @click.self="onSurfaceClick"
    @wheel="onWheel"
    @keydown="onKeydown"
  >
    <!-- The grid is a PATTERN of lines, not a background image, and it rides
         the same offset as the graph: a surface that moves under the drag is
         what makes panning legible on a canvas whose nodes are far apart. -->
    <svg
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

    <div class="absolute top-0 left-0 origin-top-left" :style="{ transform }">
      <svg
        :width="layout.width"
        :height="layout.height"
        :viewBox="`0 0 ${layout.width} ${layout.height}`"
        class="pointer-events-none absolute top-0 left-0 overflow-visible"
        aria-hidden="true"
      >
        <defs>
          <!-- One head per edge colour. `context-stroke` would collapse these
               into one marker but is still uneven across the browsers this
               site sees, and four ten-line defs cost nothing. -->
          <marker
            v-for="head in [
              { name: 'ownership', klass: 'fill-default-300' },
              { name: 'succession', klass: 'fill-secondary-300' },
              { name: 'spawn', klass: 'fill-warning-300' },
              { name: 'active', klass: 'fill-primary-500' }
            ]"
            :id="svgId(head.name)"
            :key="head.name"
            viewBox="0 0 10 10"
            refX="9"
            refY="5"
            markerWidth="5"
            markerHeight="5"
            orient="auto-start-reverse"
          >
            <path d="M 0 0 L 10 5 L 0 10 z" :class="head.klass" />
          </marker>
        </defs>

        <g
          v-for="edge in layout.edges"
          :key="edge.id"
          :class="
            cn(
              'transition-opacity duration-200',
              activeId !== null && !isEdgeLit(edge) && 'opacity-20'
            )
          "
        >
          <path
            :d="edge.path"
            fill="none"
            :stroke-width="isEdgeLit(edge) ? 2.25 : 1.5"
            :stroke-dasharray="
              edge.kind === 'succession' || edge.kind === 'spawn'
                ? '5 4'
                : undefined
            "
            :marker-end="`url(#${svgId(isEdgeLit(edge) ? 'active' : edge.kind === 'succession' ? 'succession' : edge.kind === 'spawn' ? 'spawn' : 'ownership')})`"
            :class="
              cn(
                'transition-[stroke,stroke-width] duration-200',
                isEdgeLit(edge)
                  ? 'stroke-primary-500'
                  : edge.kind === 'succession'
                    ? 'stroke-secondary-300'
                    : edge.kind === 'spawn'
                      ? 'stroke-warning-300'
                      : 'stroke-default-300'
              )
            "
          />
          <!-- The relation word appears only on the lit path. Printed on every
               edge at once it is the noise that makes a twelve-brand family
               unreadable; printed on the two or three edges you are actually
               looking at it is the whole answer. -->
          <text
            v-if="isEdgeLit(edge)"
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
        </g>
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
        :is-tab-stop="tabStopId === node.official.id"
        @pointerdown.stop
        @mouseenter="hoveredId = node.official.id"
        @mouseleave="hoveredId = null"
        @focus="focusedId = node.official.id"
        @blur="focusedId = null"
        @click="activate(node.official.id)"
        @dblclick="
          node.official.id !== currentId && emit('open', node.official.id)
        "
      />
    </div>

    <!-- Controls sit ON the canvas rather than above it: they belong to the
         view, not to the section, and a reader who has panned somewhere odd
         should not have to look away from the graph to get back. -->
    <div
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
        @click="fit"
      >
        <KunIcon name="lucide:scan" />
      </KunButton>
      <KunButton
        v-if="!isFullscreen"
        :is-icon-only="true"
        variant="flat"
        color="default"
        size="sm"
        aria-label="全屏查看"
        @click="emit('expand')"
      >
        <KunIcon name="lucide:maximize" />
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

    <div class="absolute bottom-2 left-3 flex items-center gap-2">
      <p class="text-default-400 pointer-events-none text-xs">
        滚轮缩放 · 拖拽平移 · 方向键移动 · 回车打开
      </p>
      <!-- Offered, not imposed: when the family is too big to read in a strip,
           say so where the reader is already looking instead of leaving them to
           discover the button in the corner. -->
      <KunButton
        v-if="!isFullscreen && fitScale < OFFICIAL_GRAPH_LEGIBLE_SCALE"
        variant="flat"
        color="primary"
        size="sm"
        @click.stop="emit('expand')"
      >
        <KunIcon name="lucide:maximize" />
        全屏查看
      </KunButton>
    </div>
  </div>
</template>
