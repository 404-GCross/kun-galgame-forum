<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_CONTENT_LIMIT_MAP
} from '~/constants/galgame'

const props = defineProps<{
  galgame: GalgameDetail
}>()

const emits = defineEmits<{
  onRatingCreated: [GalgameRatingCardOnGalgamePage]
}>()

const galgameAliasArray = computed(() => {
  const nameArray = Object.entries(props.galgame.name)
    .filter(
      ([_, value]) => value !== getPreferredLanguageText(props.galgame.name)
    )
    .map(([_, value]) => value)
  return nameArray.concat(props.galgame.alias)
})

const isRatingOpen = ref(false)

// "查看所有封面" modal — only worth offering when there's more than the pinned
// banner cover (covers[] includes the banner at sort_order 0).
const coversOpen = ref(false)
const hasMoreCovers = computed(() => (props.galgame.covers?.length ?? 0) > 1)
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="grid grid-cols-1 gap-3 md:grid-cols-3"
  >
    <div
      className="relative rounded-lg w-full h-full overflow-hidden md:col-span-1 aspect-video md:rounded-l-xl"
    >
      <!-- Banner is a real <KunImage>, so use the declarative
           Gallery/Item rather than the document-scan composable.
           wrap=false + v-slot lets the overlay content-limit chip
           stay a sibling that does NOT trigger the lightbox — only
           the image itself opens it. Full-res src (no `mini` variant)
           so the zoomed view is sharp. -->
      <KunLightboxGallery>
        <KunLightboxGalleryItem
          :src="getEffectiveBanner(galgame)"
          :alt="getPreferredLanguageText(galgame.name)"
          :wrap="false"
          v-slot="{ open }"
        >
          <KunImage
            class="size-full cursor-zoom-in object-cover"
            :src="getEffectiveBanner(galgame)"
            loading="eager"
            fetchpriority="high"
            :thumbhash="resolveBannerThumbhash(galgame)"
            :alt="getPreferredLanguageText(galgame.name)"
            @click="open"
          />
        </KunLightboxGalleryItem>
      </KunLightboxGallery>

      <KunChip
        variant="solid"
        class="absolute top-2 left-2"
        :color="galgame.content_limit === 'sfw' ? 'success' : 'danger'"
      >
        <KunTooltip
          position="right"
          :text="KUN_GALGAME_CONTENT_LIMIT_MAP[galgame.content_limit]"
        >
          {{ galgame.content_limit.toLocaleUpperCase() }}
        </KunTooltip>
      </KunChip>

      <!-- 查看所有封面 — sibling of the lightbox (not nested) so it doesn't trigger
           the banner lightbox; opens the covers modal. -->
      <button
        v-if="hasMoreCovers"
        type="button"
        class="bg-background/80 hover:bg-background shadow-kun-sm absolute right-2 bottom-2 z-10 inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium backdrop-blur transition-colors"
        @click="coversOpen = true"
      >
        <KunIcon name="lucide:images" class="size-4" />
        查看所有封面
      </button>
      <GalgameCovers v-model="coversOpen" :covers="galgame.covers" />
    </div>

    <div className="flex flex-col gap-3 md:col-span-2">
      <div class="flex flex-wrap items-center gap-2">
        <h1 class="text-3xl">
          {{ getPreferredLanguageText(galgame.name) }}
        </h1>
      </div>

      <div class="space-y-3">
        <KunScrollShadow
          axis="vertical"
          shadow-size="2rem"
          class-name="max-h-[100px]"
        >
          <div class="flex flex-wrap gap-2">
            <template v-for="(alias, index) in galgameAliasArray" :key="index">
              <KunChip v-if="alias">{{ alias }}</KunChip>
            </template>
          </div>
        </KunScrollShadow>

        <KunDivider />

        <div class="space-y-1 space-x-1">
          <KunChip
            v-for="(t, index) in galgame.type"
            :key="index"
            color="primary"
          >
            <KunIcon :name="GALGAME_RESOURCE_TYPE_ICON_MAP[t]" />
            {{ KUN_GALGAME_RESOURCE_TYPE_MAP[t] }}
          </KunChip>

          <KunChip
            v-for="(lang, index) in galgame.language"
            :key="index"
            color="secondary"
          >
            <KunIcon class="icon" name="lucide:globe" />
            {{ KUN_GALGAME_RESOURCE_LANGUAGE_MAP[lang] }}
          </KunChip>

          <KunChip
            v-for="(platform, index) in galgame.platform"
            :key="index"
            color="success"
          >
            <KunIcon
              class="icon"
              :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[platform]"
            />
            {{ KUN_GALGAME_RESOURCE_PLATFORM_MAP[platform] }}
          </KunChip>
        </div>

        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1">
            <!-- View count: the same compact pill as 点赞 / 收藏 (KunReaction),
                 but STATIC — action skin (no toggle), no animation, and
                 pointer-events-none so it has no hover / click effect. -->
            <!-- Hidden for not-yet-ingested wiki games: their view is always 0
                 (IncrementView no-ops without a local row), so showing it reads
                 as a misleading stat. The 未收录 notice explains the empty page. -->
            <KunReaction
              v-if="galgame.is_on_forum !== false"
              :count="galgame.view"
              :toggle="false"
              icon="lucide:eye"
              label="浏览量"
              disable-animation
              class="pointer-events-none"
            />

            <GalgameLike
              :galgame-id="galgame.id"
              :target-user-id="galgame.user.id"
              :like-count="galgame.like_count"
              :is-liked="galgame.is_liked"
            />

            <GalgameFavorite
              :galgame-id="galgame.id"
              :target-user-id="galgame.user.id"
              :favorite-count="galgame.favorite_count"
              :is-favorited="galgame.is_favorited"
            />
          </div>

          <div class="flex gap-1">
            <KunButton
              variant="shadow"
              color="primary"
              @click="isRatingOpen = true"
            >
              添加评分
            </KunButton>

            <GalgameRewrite :galgame="galgame" />

            <GalgameRatingPublish
              v-model="isRatingOpen"
              :galgame-id="galgame.id"
              @on-published="(newRating) => emits('onRatingCreated', newRating)"
            />
          </div>
        </div>
      </div>
    </div>
  </KunCard>
</template>
