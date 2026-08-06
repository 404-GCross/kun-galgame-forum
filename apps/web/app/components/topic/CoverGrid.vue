<script setup lang="ts">
import { useContentLightbox } from '@kungal/ui-vue'

// Renders a topic's covers. Each entry is a /image/<hash> content token resolved
// to an absolute CDN URL (imageTokenUrl: skips @nuxt/image IPX + the /image 302
// hop). `meta` (keyed by the same token) carries each cover's intrinsic dims +
// ThumbHash.
//
// Multi-cover layout is a SINGLE uniform-height row: every cover shares one
// height and keeps its OWN aspect ratio (width = height × ratio), so nothing is
// ever cropped and there are no letterbox bars — a tall cover (text screenshot)
// is just a narrow full-height column next to wider ones. The row never wraps;
// it scrolls horizontally when it overflows (KunScrollShadow fades the edges to
// hint there's more). Width is reserved up-front from the real dims when known
// (no reflow); otherwise the browser sizes it from the natural image on load.
const props = defineProps<{
  images: string[]
  meta?: Record<string, KunImageMeta>
  // Opt in to the fullscreen viewer (detail pages). Off in feeds — see below.
  zoomable?: boolean
}>()

const shown = computed(() => props.images.slice(0, 9))
const isSingle = computed(() => shown.value.length === 1)

const metaOf = (token: string): KunImageMeta | undefined => props.meta?.[token]

// CSS aspect-ratio from real dims so the slot reserves its width before the image
// loads (no layout shift); undefined → natural width resolves on load.
const aspectOf = (token: string): string | undefined => {
  const m = metaOf(token)
  return m?.width && m?.height ? `${m.width} / ${m.height}` : undefined
}

// The single cover is sized to the IMAGE, not to the card — a portrait cover
// stays a narrow column instead of being letterboxed into a full-width box with
// its rounded right corners stranded in the empty area. KunImage's own sizing is
// `w-full` + aspect-ratio, so the width has to come from here.
//
// It must be a DEFINITE width, not `w-auto`: a box whose width and height both
// resolve to `auto` lays out as 0×0 until the image decodes (the width/height
// attributes feed the aspect RATIO, not an intrinsic size), so nothing was
// reserved and the card jumped by the cover's full height on load — measured
// 0×0 → 682×384 in the home feed. The multi-cover row never had this: `h-40` is
// a definite height, so its ratio resolves to a width.
//
// So compute the width the browser would settle on anyway: the natural width,
// capped by the card (100%) and by the 24rem height cap. Same final box,
// reserved from the first paint. Dims-less covers keep the natural-size
// behaviour — guessing a ratio would trade this shift for a differently-wrong
// one, and a tall cover would guess worst.
const SINGLE_MAX_HEIGHT_PX = 384

const singleWidth = computed(() => {
  const m = metaOf(shown.value[0]!)
  if (!m?.width || !m?.height) {
    return undefined
  }
  const heightCapped = Math.round((SINGLE_MAX_HEIGHT_PX * m.width) / m.height)
  return { width: `min(${m.width}px, 100%, ${heightCapped}px)` }
})

// Fullscreen viewer, OPT-IN. Body images get one for free inside KunContent, but
// these covers sit outside it, so on a detail page clicking one did nothing —
// and the multi-cover row caps every cover at h-40, exactly the case where you
// most want a closer look.
//
// It must stay opt-in: in the three feed/activity cards the grid sits inside a
// KunLink to the topic, and the composable's handler preventDefaults the click.
// Attaching it there would swallow the navigation the cover exists to trigger —
// a cover in a list is the thumbnail that earns the click, not a picture to zoom.
//
// Passing a permanently-null ref is what disables it: the composable reads the
// ref once in its own onMounted, so a ref that never receives the element simply
// never gets a listener. Reading the prop at setup is fine — every call site
// passes a literal, so it never flips at runtime.
const root = ref<HTMLElement | null>(null)
const lightboxRoot = props.zoomable ? root : ref<HTMLElement | null>(null)

// The composable collects every <img> inside its container AT CLICK TIME, so the
// lightbox renders OUTSIDE that container: nested, its own full-size image and
// thumbnail strip would join the gallery on the next click. Same reason
// pm/Item.vue keeps the two siblings.
const { isLightboxOpen, images, currentImageIndex } =
  useContentLightbox(lightboxRoot)
</script>

<template>
  <div v-if="shown.length">
    <div ref="root">
      <!-- KunImage, not a bare <img> + useContentBlurUp: the composable is for
         images it does not own (markdown bodies), and it only paints the blur
         from onMounted — so a cover with its box reserved sat blank until
         hydration. KunImage owns the placeholder instead, and since ui 2.18.1
         decodes the ThumbHash during render, so the blur is already in the SSR
         HTML: blur → cross-fade to the image on load, no skeleton frame in
         between (a cover with no ThumbHash still gets the skeleton). -->
      <KunImage
        v-if="isSingle"
        :src="imageTokenUrl(shown[0]!)"
        :thumbhash="metaOf(shown[0]!)?.thumbhash"
        :aspect-ratio="aspectOf(shown[0]!)"
        :width="metaOf(shown[0]!)?.width"
        :height="metaOf(shown[0]!)?.height"
        :style="singleWidth"
        alt="话题封面"
        loading="lazy"
        object-fit="cover"
        :class-name="cn('rounded-lg', zoomable && 'cursor-zoom-in')"
      />

      <!-- Multi: one uniform-height row, each cover at its own aspect ratio (no crop,
         no bars); overflow scrolls horizontally instead of wrapping. -->
      <KunScrollShadow
        v-else
        axis="horizontal"
        shadow-size="2rem"
        scrollbar="thin"
      >
        <div class="flex gap-1.5">
          <KunImage
            v-for="(token, idx) in shown"
            :key="`${idx}-${token}`"
            :src="imageTokenUrl(token)"
            :thumbhash="metaOf(token)?.thumbhash"
            :aspect-ratio="aspectOf(token)"
            alt="话题封面"
            loading="lazy"
            object-fit="contain"
            :class-name="
              cn('h-40 w-auto shrink-0 rounded-lg', zoomable && 'cursor-zoom-in')
            "
          />
        </div>
      </KunScrollShadow>
    </div>

    <KunLightbox
      v-if="zoomable"
      v-model:is-open="isLightboxOpen"
      :images="images"
      :initial-index="currentImageIndex"
    />
  </div>
</template>
