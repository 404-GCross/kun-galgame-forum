<script setup lang="ts">
import { useContentBlurUp } from '@kungal/ui-vue'

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

// Blur-up via plain <img data-thumbhash> + the kun-ui composable. (KunImage can't
// size itself to the image's natural width, which is exactly what keeps every
// cover uncropped here — so we drive the placeholder ourselves.)
const root = ref<HTMLElement | null>(null)
useContentBlurUp(root)
</script>

<template>
  <div v-if="shown.length" ref="root">
    <!-- Single: the element is sized to the image itself (capped at the card
         width / 24rem height, ratio preserved) — NOT w-full + object-contain,
         which would letterbox a portrait into the left of a full-width box and
         leave the rounded right corners stranded in the empty area (square-right
         bug). Here the box equals the image, so all four corners round cleanly. -->
    <!-- width/height ATTRS (not just an aspect-ratio style) so the box reserves
         BEFORE load: an img with only `aspect-ratio` + `max-w-full` and no width
         has auto (=0) width until the image loads, so nothing was reserved (no
         placeholder) and the thumbhash background had 0 area to show in. The attrs
         give the browser the intrinsic size; max-w/max-h + the ratio still cap it. -->
    <img
      v-if="isSingle"
      :src="imageTokenUrl(shown[0]!)"
      :data-thumbhash="metaOf(shown[0]!)?.thumbhash || undefined"
      :width="metaOf(shown[0]!)?.width || undefined"
      :height="metaOf(shown[0]!)?.height || undefined"
      :style="
        aspectOf(shown[0]!) ? { aspectRatio: aspectOf(shown[0]!) } : undefined
      "
      alt="话题封面"
      loading="lazy"
      class="block h-auto max-h-96 w-auto max-w-full rounded-lg"
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
        <img
          v-for="(token, idx) in shown"
          :key="`${idx}-${token}`"
          :src="imageTokenUrl(token)"
          :data-thumbhash="metaOf(token)?.thumbhash || undefined"
          :style="
            aspectOf(token) ? { aspectRatio: aspectOf(token) } : undefined
          "
          alt="话题封面"
          loading="lazy"
          class="h-40 w-auto shrink-0 rounded-lg object-contain"
        />
      </div>
    </KunScrollShadow>
  </div>
</template>
