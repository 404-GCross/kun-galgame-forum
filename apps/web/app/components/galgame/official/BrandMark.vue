<script setup lang="ts">
// The 会社's own brand mark, shared by the maker's two pages so the identity is
// literally the same object on both — large on the overview, smaller above the
// games grid where the title carries the page instead.
//
// Always on its own light plate, never bare: a logo is usually a transparent
// PNG drawn in one dark colour, and against the dark theme's background half of
// them disappear entirely. Logos are SFW upstream, so no content gate sits in
// front of it.
withDefaults(
  defineProps<{
    src: string
    name: string
    size?: 'md' | 'lg'
  }>(),
  { size: 'lg' }
)
</script>

<template>
  <div class="bg-default-100 shrink-0 rounded-lg p-2 sm:p-3">
    <KunImage
      :src="src"
      :alt="`${name} logo`"
      loading="eager"
      object-fit="contain"
      :class-name="
        cn(size === 'lg' ? 'size-14 sm:size-20' : 'size-10 sm:size-14')
      "
    />
  </div>
</template>
