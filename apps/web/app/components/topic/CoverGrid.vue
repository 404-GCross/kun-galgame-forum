<script setup lang="ts">
// Renders a topic's covers. Each entry is a /image/<hash> content token resolved
// to an absolute CDN URL (imageTokenUrl: skips @nuxt/image IPX + the /image 302
// hop). `meta` (keyed by the same token) carries each cover's intrinsic dims +
// ThumbHash, so the image reserves its aspect ratio (no CLS) and blurs up while
// loading. Images keep their ORIGINAL aspect ratio and are never cropped
// (object-contain); height is only capped so a tall image can't dominate.
const props = defineProps<{
  images: string[]
  meta?: Record<string, KunImageMeta>
}>()

const shown = computed(() => props.images.slice(0, 9))
const isSingle = computed(() => shown.value.length === 1)
// 2/4 = 2-col, everything else = 3-col (single is rendered on its own below).
const gridClass = computed(() =>
  shown.value.length === 2 || shown.value.length === 4
    ? 'grid-cols-2'
    : 'grid-cols-3'
)

const metaOf = (token: string): KunImageMeta | undefined => props.meta?.[token]
// Only reserve the single cover's box when we know its real dims — otherwise let
// it keep its natural (capped) height, exactly as before (no pre-backfill regression).
const singleAspect = computed(() => {
  const m = metaOf(shown.value[0]!)
  return m?.width && m?.height ? `${m.width} / ${m.height}` : undefined
})
</script>

<template>
  <div v-if="shown.length">
    <!-- Single: at the image's own ratio (no crop), just cap the height. A tall
         image is letterboxed within the full width — keep it LEFT-aligned
         (object-left), never centered. -->
    <KunImage
      v-if="isSingle"
      :src="imageTokenUrl(shown[0]!)"
      alt="话题封面"
      loading="lazy"
      object-fit="contain"
      :aspect-ratio="singleAspect"
      :thumbhash="metaOf(shown[0]!)?.thumbhash"
      class="bg-default-100 max-h-96 w-full rounded-lg object-contain object-left"
    />

    <!-- Multi: uniform grid; object-contain keeps each image's ratio (no crop),
         letterboxed in the cell against the neutral card fill. -->
    <div v-else :class="cn('grid w-full gap-1.5', gridClass)">
      <div
        v-for="(token, idx) in shown"
        :key="`${idx}-${token}`"
        class="bg-default-100 h-40 overflow-hidden rounded-lg"
      >
        <KunImage
          :src="imageTokenUrl(token)"
          alt="话题封面"
          loading="lazy"
          object-fit="contain"
          :thumbhash="metaOf(token)?.thumbhash"
          class="h-full w-full object-contain"
        />
      </div>
    </div>
  </div>
</template>
