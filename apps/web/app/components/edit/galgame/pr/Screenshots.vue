<script setup lang="ts">
// Screenshot gallery editor (U2 K-PR3b) — image-first grid.
//
// Each row references an image_service hash; rows are ordered by sort_order.
// Distinct from covers — no pinned/banner concept, just an ordered gallery
// with optional captions. Upload uses the galgame_screenshot preset (main
// image only — no unused variants), multi-file/drag via useGalgameImageUpload.
// Reorder swaps sort_order with a neighbour (no drag-and-drop in v1). Caption +
// ratings live in each tile's ⚙ popover so the grid stays image-first instead
// of a wall of always-open forms. Footer.vue strips `cdn_url` (preview field).
const { galgamePR } = storeToRefs(useTempGalgamePRStore())

const screenshots = computed<GalgameScreenshot[]>({
  get: () => galgamePR.value[0]?.screenshots ?? [],
  set: (v) => {
    if (galgamePR.value[0]) galgamePR.value[0].screenshots = v
  }
})

const sorted = computed(() =>
  [...screenshots.value].sort((a, b) => {
    if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order
    return a.image_hash.localeCompare(b.image_hash)
  })
)

const { isUploading, progressText, uploadFiles } = useGalgameImageUpload({
  preset: 'galgame_screenshot',
  rows: screenshots,
  dedupeLabel: '截图',
  makeRow: (res, sortOrder) => ({
    image_hash: res.hash,
    sort_order: sortOrder,
    caption: '',
    sexual: 0,
    violence: 0,
    source: '',
    source_key: '',
    cdn_url: res.url
  })
})

const handleRemove = (hash: string) => {
  screenshots.value = screenshots.value.filter((s) => s.image_hash !== hash)
}

// Move earlier (-1) / later (+1) by swapping sort_order with the neighbour in
// display order. Grid reads left→right, top→bottom, so chevron-left = 前移.
const handleShift = (target: GalgameScreenshot, dir: -1 | 1) => {
  const idx = sorted.value.findIndex((s) => s.image_hash === target.image_hash)
  const neighbour = sorted.value[idx + dir]
  if (!neighbour) return
  const a = target.sort_order
  const b = neighbour.sort_order
  screenshots.value = screenshots.value.map((s) => {
    if (s.image_hash === target.image_hash) return { ...s, sort_order: b }
    if (s.image_hash === neighbour.image_hash) return { ...s, sort_order: a }
    return s
  })
}
</script>

<template>
  <div class="space-y-3">
    <h2 class="text-xl">画廊 / 截图</h2>
    <p class="text-default-500 text-sm">
      游戏截图 / CG / 立绘等。用箭头调整顺序,⚙
      可填写说明与分级。提交后整组替换。
    </p>

    <KunLightboxGallery>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
        <div
          v-for="(shot, idx) in sorted"
          :key="shot.image_hash"
          class="group border-default-200 relative overflow-hidden rounded-lg border"
        >
          <KunLightboxGalleryItem
            :src="galgameImageSrc(shot)"
            :alt="shot.caption || ''"
            :wrap="false"
            v-slot="{ open }"
          >
            <KunImage
              :src="galgameImageSrc(shot)"
              :alt="shot.caption || ''"
              loading="lazy"
              :thumbhash="shot.thumbhash"
              class-name="aspect-video w-full cursor-zoom-in object-cover"
              @click="open"
            />
          </KunLightboxGalleryItem>

          <!-- per-tile toolbar: visible on touch, hover-revealed on desktop -->
          <div
            class="absolute top-1.5 right-1.5 flex gap-1 opacity-100 transition-opacity sm:opacity-0 sm:group-focus-within:opacity-100 sm:group-hover:opacity-100"
          >
            <KunTooltip text="前移">
              <KunButton
                :is-icon-only="true"
                size="sm"
                variant="solid"
                color="default"
                :disabled="idx === 0"
                @click="handleShift(shot, -1)"
              >
                <KunIcon name="lucide:chevron-left" />
              </KunButton>
            </KunTooltip>
            <KunTooltip text="后移">
              <KunButton
                :is-icon-only="true"
                size="sm"
                variant="solid"
                color="default"
                :disabled="idx === sorted.length - 1"
                @click="handleShift(shot, 1)"
              >
                <KunIcon name="lucide:chevron-right" />
              </KunButton>
            </KunTooltip>

            <KunPopover position="bottom-end" inner-class="p-3 w-64">
              <template #trigger>
                <KunButton
                  :is-icon-only="true"
                  size="sm"
                  variant="solid"
                  color="default"
                  aria-label="说明与分级"
                >
                  <KunIcon name="lucide:settings-2" />
                </KunButton>
              </template>
              <div class="space-y-2">
                <KunInput
                  v-model="shot.caption"
                  placeholder="说明文字 (可选)"
                />
                <EditGalgamePrImageRatingFields
                  v-model:sexual="shot.sexual"
                  v-model:violence="shot.violence"
                />
              </div>
            </KunPopover>

            <KunTooltip text="移除">
              <KunButton
                :is-icon-only="true"
                size="sm"
                variant="solid"
                color="danger"
                @click="handleRemove(shot.image_hash)"
              >
                <KunIcon name="lucide:trash-2" />
              </KunButton>
            </KunTooltip>
          </div>

          <!-- rating badges (only when rated) -->
          <div
            v-if="shot.sexual || shot.violence"
            class="pointer-events-none absolute top-1.5 left-1.5 flex gap-1"
          >
            <KunChip
              v-if="shot.sexual"
              size="sm"
              color="warning"
              variant="solid"
            >
              性 {{ shot.sexual }}
            </KunChip>
            <KunChip
              v-if="shot.violence"
              size="sm"
              color="danger"
              variant="solid"
            >
              暴 {{ shot.violence }}
            </KunChip>
          </div>

          <!-- caption preview strip -->
          <div
            v-if="shot.caption"
            class="pointer-events-none absolute right-0 bottom-0 left-0 truncate bg-black/55 px-2 py-1 text-xs text-white"
          >
            {{ shot.caption }}
          </div>
        </div>

        <EditGalgamePrImageDropzone
          label="添加截图"
          :is-uploading="isUploading"
          :progress-text="progressText"
          @files="uploadFiles"
        />
      </div>
    </KunLightboxGallery>
  </div>
</template>
