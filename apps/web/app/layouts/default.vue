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
      <!-- No transition on this wrapper, deliberately. Its box is pinned by
           max-w-7xl and two desktop-nav: margins, and desktop-nav is a pure
           media query, so nothing here changes at runtime and a transition
           could never animate anything intended. What it did animate was
           accidents: any stray reflow of the page column (an overlay's
           scrollbar compensation, a late-arriving tab) stopped being an
           instant, near-invisible jump and became a 300ms glide of the whole
           content column, seen as the cards inside it shaking. Layout shifts
           are worth removing; animating them is not. -->
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
        <!-- Small gutter everywhere except the desktop rail layout (desktop-nav),
             which is flush. pt clears the fixed top bar. -->
        <div class="desktop-nav:px-0 h-full px-2 pt-22 pb-6">
          <NuxtPage />
        </div>

        <!-- Native <img>, NOT KunImage: this mascot must stay `fixed` to the
             bottom-right corner. KunImage's skeleton wrapper is a
             `position: relative` div that also receives this class, and
             Tailwind emits `.relative` after `.fixed`, so `relative` wins and
             un-pins it. A decorative static webp also wants no skeleton / IPX
             round-trip, so a bare <img> is the right tool. -->
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
