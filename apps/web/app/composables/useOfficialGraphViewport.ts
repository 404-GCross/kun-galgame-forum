import { useElementSize } from '@vueuse/core'
import type { Ref, ComputedRef } from 'vue'

// Pan and zoom for the 会社 relation graph: one transform, and every gesture
// that can move it.
//
// Separated from the canvas component for the ordinary reason — the canvas is
// already carrying selection, highlighting and keyboard navigation — but also
// because this is the half with no markup at all, which makes the arithmetic
// readable on its own.
//
// The rule the whole thing turns on: zooming must keep the point under the
// cursor under the cursor. Scaling around the container's origin instead is
// the single most common way a graph viewport ends up feeling broken — the
// thing you are looking at slides off screen as you zoom toward it.

const MIN_SCALE = 0.4
const MAX_SCALE = 1.8
/** Breathing room around a fitted graph, in screen pixels. */
const FIT_PADDING = 28
/** Below this, brand names stop being readable and "fit everything" stops being
 * a service to anyone. */
export const OFFICIAL_GRAPH_LEGIBLE_SCALE = 0.62
/** How much of a node's own size to keep clear when panning it into view. */
const REVEAL_MARGIN = 24

const clamp = (value: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, value))

export const useOfficialGraphViewport = (
  viewport: Ref<HTMLElement | null>,
  content: ComputedRef<{ width: number; height: number }>,
  /** Where to look when the whole graph will not fit legibly. */
  focus?: ComputedRef<{ x: number; y: number } | null>
) => {
  const { width: viewW, height: viewH } = useElementSize(viewport)

  const scale = ref(1)
  const offsetX = ref(0)
  const offsetY = ref(0)
  const isPanning = ref(false)

  /** The scale that would show everything. */
  const fitScale = computed(() => {
    const { width, height } = content.value
    if (!width || !height || !viewW.value || !viewH.value) return 1
    return clamp(
      Math.min(
        (viewW.value - FIT_PADDING * 2) / width,
        (viewH.value - FIT_PADDING * 2) / height,
        1
      ),
      MIN_SCALE,
      MAX_SCALE
    )
  })

  /** Frame the whole graph. Never zooms IN past 1 — a two-node family blown up
   * to fill a 420px canvas reads as a mistake, not as emphasis. */
  const fit = () => {
    const { width, height } = content.value
    if (!width || !height || !viewW.value || !viewH.value) return
    const k = fitScale.value
    scale.value = k
    offsetX.value = (viewW.value - width * k) / 2
    offsetY.value = (viewH.value - height * k) / 2
  }

  /** Zoom about a point given in VIEWPORT coordinates. */
  const zoomAt = (factor: number, atX: number, atY: number) => {
    const next = clamp(scale.value * factor, MIN_SCALE, MAX_SCALE)
    const ratio = next / scale.value
    offsetX.value = atX - (atX - offsetX.value) * ratio
    offsetY.value = atY - (atY - offsetY.value) * ratio
    scale.value = next
  }

  const zoomBy = (factor: number) =>
    zoomAt(factor, viewW.value / 2, viewH.value / 2)

  /** Put a CONTENT-space point in the middle of the viewport. */
  const centerOn = (x: number, y: number) => {
    offsetX.value = viewW.value / 2 - x * scale.value
    offsetY.value = viewH.value / 2 - y * scale.value
  }

  /**
   * The opening view — NOT simply "fit".
   *
   * A family that only fits at 0.42 fits as unreadable confetti: the reader
   * arrives at a picture whose whole content is "there is a lot of this". So
   * below a legible scale the graph opens ON the 会社 the page is about, at a
   * size where the names can be read, and 适应画布 is one button away for
   * whoever wants the whole shape.
   */
  const frame = () => {
    if (fitScale.value >= OFFICIAL_GRAPH_LEGIBLE_SCALE || !focus?.value) {
      fit()
      return
    }
    scale.value = OFFICIAL_GRAPH_LEGIBLE_SCALE
    centerOn(focus.value.x, focus.value.y)
  }

  /** Pan the minimum distance that brings a content-space box fully on screen —
   * what keyboard navigation needs, and only that. Recentring on every arrow
   * key would drag the whole picture under the reader on each step. */
  const reveal = (x: number, y: number, width: number, height: number) => {
    const left = offsetX.value + (x - width / 2) * scale.value
    const top = offsetY.value + (y - height / 2) * scale.value
    const right = left + width * scale.value
    const bottom = top + height * scale.value

    if (left < REVEAL_MARGIN) offsetX.value += REVEAL_MARGIN - left
    else if (right > viewW.value - REVEAL_MARGIN)
      offsetX.value -= right - (viewW.value - REVEAL_MARGIN)

    if (top < REVEAL_MARGIN) offsetY.value += REVEAL_MARGIN - top
    else if (bottom > viewH.value - REVEAL_MARGIN)
      offsetY.value -= bottom - (viewH.value - REVEAL_MARGIN)
  }

  // Pointer bookkeeping. Two live pointers is a pinch; one is a drag.
  const pointers = new Map<number, { x: number; y: number }>()
  let pinchDistance = 0

  const localPoint = (event: PointerEvent) => {
    const box = viewport.value?.getBoundingClientRect()
    return {
      x: event.clientX - (box?.left ?? 0),
      y: event.clientY - (box?.top ?? 0)
    }
  }

  const onPointerDown = (event: PointerEvent) => {
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })
    if (pointers.size === 1) isPanning.value = true
    if (pointers.size === 2) pinchDistance = 0
    ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  }

  const onPointerMove = (event: PointerEvent) => {
    const previous = pointers.get(event.pointerId)
    if (!previous) return
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })

    if (pointers.size >= 2) {
      const [a, b] = [...pointers.values()]
      if (!a || !b) return
      const distance = Math.hypot(a.x - b.x, a.y - b.y)
      const box = viewport.value?.getBoundingClientRect()
      const midX = (a.x + b.x) / 2 - (box?.left ?? 0)
      const midY = (a.y + b.y) / 2 - (box?.top ?? 0)
      if (pinchDistance > 0 && distance > 0) {
        zoomAt(distance / pinchDistance, midX, midY)
      }
      pinchDistance = distance
      return
    }

    offsetX.value += event.clientX - previous.x
    // A finger dragging up the page is scrolling the page, not the graph:
    // `touch-action: pan-y` leaves the vertical axis to the browser, so
    // claiming it here would fight the scroll the reader actually asked for.
    if (event.pointerType === 'mouse') {
      offsetY.value += event.clientY - previous.y
    }
  }

  const onPointerUp = (event: PointerEvent) => {
    pointers.delete(event.pointerId)
    if (pointers.size < 2) pinchDistance = 0
    if (!pointers.size) isPanning.value = false
  }

  /**
   * The wheel zooms. That is what a canvas is for, and asking for a modifier
   * key first makes the obvious gesture do nothing.
   *
   * What keeps it from being a scroll trap is the LIMIT: once the graph cannot
   * zoom any further in the direction being asked for, the event is left alone
   * and the page scrolls on past. So a reader who spins the wheel over the
   * graph zooms out to the whole family and then keeps going down the page,
   * without ever finding out there was a rule.
   */
  const onWheel = (event: WheelEvent) => {
    const zoomingIn = event.deltaY < 0
    const stuck = zoomingIn
      ? scale.value >= MAX_SCALE - 0.001
      : scale.value <= MIN_SCALE + 0.001
    if (stuck) return
    event.preventDefault()
    const { x, y } = localPoint(event as unknown as PointerEvent)
    // Trackpads deliver many small deltas and a mouse wheel a few large ones;
    // scaling the step by the delta keeps both feeling like the same gesture.
    const strength = Math.min(Math.abs(event.deltaY) / 100, 2)
    zoomAt(zoomingIn ? 1 + 0.14 * strength : 1 / (1 + 0.14 * strength), x, y)
  }

  const transform = computed(
    () =>
      `translate(${offsetX.value}px, ${offsetY.value}px) scale(${scale.value})`
  )

  // Re-frame when the graph or the box it lives in changes size — a sidebar
  // opening or a phone rotating must not leave the drawing off screen.
  watch([() => content.value.width, viewW, viewH], frame, { immediate: true })

  return {
    scale,
    fitScale,
    offsetX,
    offsetY,
    isPanning,
    transform,
    fit,
    frame,
    zoomBy,
    centerOn,
    reveal,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    onWheel
  }
}
