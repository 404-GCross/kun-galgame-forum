<script setup lang="ts">
import {
  ENABLE_KUN_VISUAL_NOVEL_FORUM_WINTER_THEME,
  KUN_VISUAL_NOVEL_FORUM_WINTER_THEME_BACKGROUND
} from '~/config/theme'

const {
  showKUNGalgameBackground,
  showKUNGalgameBackLoli,
  showKUNGalgameBackgroundOpacity
} = storeToRefs(usePersistSettingsStore())

const imageURL = ref(
  ENABLE_KUN_VISUAL_NOVEL_FORUM_WINTER_THEME
    ? KUN_VISUAL_NOVEL_FORUM_WINTER_THEME_BACKGROUND
    : ''
)

onMounted(async () => {
  imageURL.value = await usePersistSettingsStore().getCurrentBackground()
})

watch(
  () => showKUNGalgameBackground.value,
  async () => {
    imageURL.value = await usePersistSettingsStore().getCurrentBackground()
  }
)
</script>

<template>
  <div class="contents">
    <div class="bg-background fixed top-0 left-0 h-full w-full">
      <div
        class="fixed size-full bg-cover bg-fixed bg-center bg-no-repeat brightness-[var(--kun-background-brightness)]"
        :style="{
          backgroundImage: `url(${imageURL})`,
          opacity: showKUNGalgameBackgroundOpacity / 100
        }"
      />
    </div>

    <div class="desktop-nav:block hidden">
      <KunLayoutSidebar />
    </div>

    <KunTopBar />

    <div class="bg-background flex min-h-dvh min-h-screen justify-center">
      <div
        :class="
          cn(
            'desktop-nav:mr-3 z-10 w-full max-w-7xl min-w-0',
            // Clears the desktop icon rail (w-20 = 80px) PLUS a ~24px gap so page
            // content (e.g. the home tab rail) isn't flush against it. Only when the
            // rail is actually shown (desktop-nav) — touch tablets get no rail, so
            // no offset; the hamburger drawer overlays instead.
            'desktop-nav:ml-[104px]'
          )
        "
      >
        <div class="desktop-nav:px-0 h-full px-2 pt-22 pb-6">
          <NuxtPage />
        </div>

        <img
          v-if="showKUNGalgameBackLoli"
          class="pointer-events-none fixed right-px bottom-px z-0 h-[33dvh] w-auto opacity-17! select-none"
          :src="
            ENABLE_KUN_VISUAL_NOVEL_FORUM_WINTER_THEME
              ? '/winter/sd.webp'
              : '/image/kohaku.webp'
          "
          loading="lazy"
          alt="kohaku"
        />
      </div>
    </div>
  </div>
</template>
