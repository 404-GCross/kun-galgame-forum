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
// One gesture per level of commitment, so nobody navigates away from the page
// by brushing past a box.
//
// Pointer only, deliberately. The graph is a picture you point at; the 列表 tab
// is the same family as text, and that is the reading a keyboard, a screen
// reader and a JS-less render all get — a better one than a canvas driven by
// arrow keys.
import { useMediaQuery } from '@vueuse/core'
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

/**
 * On a phone, in the page, this is a PREVIEW — not a small canvas.
 *
 * A 22rem strip cannot host a pan-and-zoom surface that also has to share the
 * page's vertical scroll: whichever axis the graph takes, the reader loses. So
 * on a narrow screen the inline graph shows the whole shape, takes no gestures
 * at all, and a tap anywhere opens the fullscreen view, where the graph owns
 * the screen and every gesture is unambiguous. Wide screens and fullscreen are
 * the real thing.
 */
const isWide = useMediaQuery('(min-width: 640px)')
const isInteractive = computed(() => props.isFullscreen || isWide.value)
const isPreview = computed(() => !isInteractive.value)

// Fullscreen owns the screen, so a finger there drags the graph in both axes
// and pinch is ours to handle; in the page the vertical axis stays the page's.
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

// The graph opens with the current 会社 selected so the panel has something to
// say — but a selection made FOR the reader must not dim the picture they came
// to look at. Until they point at something themselves, nothing steps back.
const hasPointed = ref(false)
watch([selectedId, hoveredId], () => {
  hasPointed.value = true
})

// What the highlight follows, most immediate intent first. The preview has no
// pointer at all, so there it always says "you are here" — at thumbnail scale
// one lit box among grey ones is the only thing that survives being small.
const activeId = computed(() => {
  if (isPreview.value) return props.currentId
  if (hoveredId.value !== null) return hoveredId.value
  if (!hasPointed.value) return null
  return selectedId.value
})

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
  // Nothing steps back in a thumbnail — at that scale dimming is the difference
  // between a faint shape and no shape. The current 会社's edges still light, so
  // the eye lands on it.
  isPreview.value ||
  activeId.value === null ||
  activeId.value === id ||
  (neighbours.value.get(activeId.value)?.has(id) ?? false)

const isEdgeLit = (edge: GalgameOfficialGraphEdge) =>
  activeId.value !== null &&
  (edge.from === activeId.value || edge.to === activeId.value)

/** Select, then open: the second activation of the same node is the one that
 * leaves the page. The 会社 whose page this is can never be opened — a link
 * back to where you already are is the one navigation a graph must not offer. */
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

/** One colour per edge, named twice because the line strokes it and the head
 * fills it — and they are drawn in different layers, so they cannot share a
 * `<g>` and inherit it. */
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

// Clicking the empty canvas clears the selection — but only when it was a
// click and not the end of a drag, which is the difference between dismissing
// the panel and losing it every time you pan.
const DRAG_SLOP = 4
let pressedAt = { x: 0, y: 0 }

const onSurfacePointerDown = (event: PointerEvent) => {
  if (isPreview.value) return
  pressedAt = { x: event.clientX, y: event.clientY }
  onPointerDown(event)
}

const onSurfaceClick = (event: MouseEvent) => {
  // A preview takes exactly one gesture, and it is the one that gets you out of
  // the preview.
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

// A preview never eats the page's scroll, so the wheel passes straight through.
const onSurfaceWheel = (event: WheelEvent) => {
  if (isPreview.value) return
  onWheel(event)
}

// Marker ids must be unique per instance: two graphs on one page sharing an id
// would both resolve to whichever <defs> the document happened to parse first.
const uid = useId()
const svgId = (name: string) => `kun-official-graph-${name}-${uid}`
</script>

<template>
  <div
    ref="viewport"
    :aria-label="
      isPreview ? '会社关系图预览, 点击全屏查看' : '会社关系图, 拖拽平移, 滚轮缩放'
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
    <!-- The grid is a PATTERN of lines, not a background image, and it rides
         the same offset as the graph: a surface that moves under the drag is
         what makes panning legible on a canvas whose nodes are far apart. -->
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

    <!-- In preview the boxes are a picture, not controls: no hover, no focus,
         no tap of their own — the single gesture belongs to the surface. -->
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
        <!-- Three layers, and the order is the point: every line, then every
             head, then the labels. Drawn edge by edge instead, a line crossing
             near a box is painted after the arrow that already landed there and
             takes the tip off it. -->
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

        <!-- The head as a swept-back chevron rather than a plain triangle: its
             notch sits exactly where the line stops, so the two read as one
             stroke that sharpens instead of a wedge parked on the end. -->
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

        <!-- The relation word appears only on the lit path. Printed on every
             edge at once it is the noise that makes a twelve-brand family
             unreadable; printed on the two or three edges you are actually
             looking at it is the whole answer. -->
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

    <!-- Controls sit ON the canvas rather than above it: they belong to the
         view, not to the section, and a reader who has panned somewhere odd
         should not have to look away from the graph to get back. -->
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

    <!-- The one way into fullscreen, at the bottom where the reader is already
         looking. It used to be duplicated as an icon in the corner column,
         which is two answers to one question. -->
    <div
      class="absolute right-3 bottom-2 left-3 flex items-center gap-2"
      @pointerdown.stop
    >
      <!-- The hint names the gestures this surface actually HAS. Telling a
           phone reader about the scroll wheel is noise; telling them to pinch a
           preview that takes no gestures is a lie. -->
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
