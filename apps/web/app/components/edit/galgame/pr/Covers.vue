<script setup lang="ts">
// Cover candidate-set editor (U2 K-PR3b) — image-first grid.
//
// covers[sort_order=0] is the pinned banner (= effective banner head image);
// other rows are alternate candidates. Pin = make sort_order=0; pinCoverAtomic
// atomically demotes the old pinned row so the wiki's partial unique index is
// never violated on submit. Upload (galgame_banner preset, multi-file/drag via
// useGalgameImageUpload) appends rows; the whole set replaces the galgame's
// covers on submit. Footer.vue strips `cdn_url` (a render-only preview field)
// from the wire payload — the wiki doesn't accept it on write.
const { galgamePR } = storeToRefs(useTempGalgamePRStore())

const covers = computed<GalgameCover[]>({
  get: () => galgamePR.value[0]?.covers ?? [],
  set: (v) => {
    if (galgamePR.value[0]) galgamePR.value[0].covers = v
  }
})

const sorted = computed(() =>
  [...covers.value].sort((a, b) => {
    if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order
    return a.image_hash.localeCompare(b.image_hash)
  })
)

const { isUploading, progressText, uploadFiles } = useGalgameImageUpload({
  preset: 'galgame_banner',
  rows: covers,
  dedupeLabel: '封面',
  makeRow: (res, sortOrder) => ({
    image_hash: res.hash,
    sort_order: sortOrder,
    sexual: 0,
    violence: 0,
    source: '',
    source_key: '',
    cdn_url: res.url
  })
})

// Atomic pin: picked → 0, old 0-row → (max+1). Logic + edge cases in
// shared/utils/pinCoverAtomic.ts (covered by pinCoverAtomic.spec.ts).
const handlePin = (target: GalgameCover) => {
  covers.value = pinCoverAtomic(covers.value, target.image_hash)
}

const handleRemove = (hash: string) => {
  covers.value = covers.value.filter((c) => c.image_hash !== hash)
}
</script>

<template>
  <div class="space-y-3">
    <h2 class="text-xl">封面</h2>
    <p class="text-default-500 text-sm">
      添加候选封面图;其中 <span class="text-primary">钉为封面</span>
      的一张作为详情页头图,其余为备选。提交后整组替换该 Galgame 的封面集。
    </p>

    <KunLightboxGallery>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
        <div
          v-for="cover in sorted"
          :key="cover.image_hash"
          class="group border-default-200 relative overflow-hidden rounded-lg border"
          :class="{
            'ring-primary border-primary ring-2': cover.sort_order === 0
          }"
        >
          <KunLightboxGalleryItem
            :src="galgameImageSrc(cover)"
            :wrap="false"
            v-slot="{ open }"
          >
            <KunImage
              :src="galgameImageSrc(cover)"
              loading="lazy"
              :thumbhash="cover.thumbhash"
              class-name="aspect-video w-full cursor-zoom-in object-cover"
              @click="open"
            />
          </KunLightboxGalleryItem>

          <!-- pinned banner badge (top-left) -->
          <KunChip
            v-if="cover.sort_order === 0"
            color="primary"
            variant="solid"
            size="sm"
            class="pointer-events-none absolute top-1.5 left-1.5"
          >
            <KunIcon name="lucide:pin" class="mr-1" />封面
          </KunChip>

          <!-- per-tile toolbar: visible on touch, hover-revealed on desktop -->
          <div
            class="absolute top-1.5 right-1.5 flex gap-1 opacity-100 transition-opacity sm:opacity-0 sm:group-focus-within:opacity-100 sm:group-hover:opacity-100"
          >
            <KunTooltip v-if="cover.sort_order !== 0" text="钉为封面">
              <KunButton
                :is-icon-only="true"
                size="sm"
                variant="solid"
                color="default"
                @click="handlePin(cover)"
              >
                <KunIcon name="lucide:pin" />
              </KunButton>
            </KunTooltip>

            <KunPopover position="bottom-end" inner-class="p-3 w-56">
              <template #trigger>
                <KunButton
                  :is-icon-only="true"
                  size="sm"
                  variant="solid"
                  color="default"
                  aria-label="评级"
                >
                  <KunIcon name="lucide:settings-2" />
                </KunButton>
              </template>
              <EditGalgamePrImageRatingFields
                v-model:sexual="cover.sexual"
                v-model:violence="cover.violence"
              />
            </KunPopover>

            <KunTooltip text="移除">
              <KunButton
                :is-icon-only="true"
                size="sm"
                variant="solid"
                color="danger"
                @click="handleRemove(cover.image_hash)"
              >
                <KunIcon name="lucide:trash-2" />
              </KunButton>
            </KunTooltip>
          </div>

          <!-- rating badges (only when rated) -->
          <div
            v-if="cover.sexual || cover.violence"
            class="pointer-events-none absolute bottom-1.5 left-1.5 flex gap-1"
          >
            <KunChip
              v-if="cover.sexual"
              size="sm"
              color="warning"
              variant="solid"
            >
              性 {{ cover.sexual }}
            </KunChip>
            <KunChip
              v-if="cover.violence"
              size="sm"
              color="danger"
              variant="solid"
            >
              暴 {{ cover.violence }}
            </KunChip>
          </div>
        </div>

        <EditGalgamePrImageDropzone
          label="添加封面"
          :is-uploading="isUploading"
          :progress-text="progressText"
          @files="uploadFiles"
        />
      </div>
    </KunLightboxGallery>
  </div>
</template>
