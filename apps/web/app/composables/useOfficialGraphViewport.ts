import { useElementSize } from '@vueuse/core'
import type { Ref, ComputedRef } from 'vue'

const MIN_SCALE = 0.4
const MAX_SCALE = 1.8
const FIT_PADDING = 28
export const OFFICIAL_GRAPH_LEGIBLE_SCALE = 0.62

const clamp = (value: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, value))

interface OfficialGraphViewportOptions {
  focus?: ComputedRef<{ x: number; y: number } | null>
  freeTouchPan?: ComputedRef<boolean>
  preview?: ComputedRef<boolean>
}

export const useOfficialGraphViewport = (
  viewport: Ref<HTMLElement | null>,
  content: ComputedRef<{ width: number; height: number }>,
  options: OfficialGraphViewportOptions = {}
) => {
  const { focus, freeTouchPan, preview } = options
  const { width: viewW, height: viewH } = useElementSize(viewport)

  const scale = ref(1)
  const offsetX = ref(0)
  const offsetY = ref(0)
  const isPanning = ref(false)

  const rawFitScale = computed(() => {
    const { width, height } = content.value
    if (!width || !height || !viewW.value || !viewH.value) return 1
    return Math.min(
      (viewW.value - FIT_PADDING * 2) / width,
      (viewH.value - FIT_PADDING * 2) / height,
      1
    )
  })

  const fitScale = computed(() =>
    clamp(rawFitScale.value, MIN_SCALE, MAX_SCALE)
  )

  const fit = (allowTiny = false) => {
    const { width, height } = content.value
    if (!width || !height || !viewW.value || !viewH.value) return
    const k = allowTiny ? rawFitScale.value : fitScale.value
    scale.value = k
    offsetX.value = (viewW.value - width * k) / 2
    offsetY.value = (viewH.value - height * k) / 2
  }

  const zoomAt = (factor: number, atX: number, atY: number) => {
    const next = clamp(scale.value * factor, MIN_SCALE, MAX_SCALE)
    const ratio = next / scale.value
    offsetX.value = atX - (atX - offsetX.value) * ratio
    offsetY.value = atY - (atY - offsetY.value) * ratio
    scale.value = next
  }

  const zoomBy = (factor: number) =>
    zoomAt(factor, viewW.value / 2, viewH.value / 2)

  const centerOn = (x: number, y: number) => {
    offsetX.value = viewW.value / 2 - x * scale.value
    offsetY.value = viewH.value / 2 - y * scale.value
  }

  const frame = () => {
    if (preview?.value) {
      fit(true)
      return
    }
    if (fitScale.value >= OFFICIAL_GRAPH_LEGIBLE_SCALE || !focus?.value) {
      fit()
      return
    }
    scale.value = OFFICIAL_GRAPH_LEGIBLE_SCALE
    centerOn(focus.value.x, focus.value.y)
  }

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
    if (event.pointerType === 'mouse' || freeTouchPan?.value) {
      offsetY.value += event.clientY - previous.y
    }
  }

  const onPointerUp = (event: PointerEvent) => {
    pointers.delete(event.pointerId)
    if (pointers.size < 2) pinchDistance = 0
    if (!pointers.size) isPanning.value = false
  }

  const onWheel = (event: WheelEvent) => {
    const zoomingIn = event.deltaY < 0
    const stuck = zoomingIn
      ? scale.value >= MAX_SCALE - 0.001
      : scale.value <= MIN_SCALE + 0.001
    if (stuck) return
    event.preventDefault()
    const { x, y } = localPoint(event as unknown as PointerEvent)
    const strength = Math.min(Math.abs(event.deltaY) / 100, 2)
    zoomAt(zoomingIn ? 1 + 0.14 * strength : 1 / (1 + 0.14 * strength), x, y)
  }

  const transform = computed(
    () =>
      `translate(${offsetX.value}px, ${offsetY.value}px) scale(${scale.value})`
  )

  watch(
    [() => content.value.width, viewW, viewH, () => preview?.value],
    frame,
    {
      immediate: true
    }
  )

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
    onPointerDown,
    onPointerMove,
    onPointerUp,
    onWheel
  }
}
