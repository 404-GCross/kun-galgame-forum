<script setup lang="ts">
// The attribution line under a game's title on a rolled-up 会社 page.
//
// A holding company's page lists its imprints' games as well as its own, which
// is the only way most of these pages have anything on them at all. This is
// what keeps that from being a lie: the row is on VISUAL ARTS' page, the game
// is Key's, and the reader is told which.
//
// Deliberately not a link, even though the imprint has a page of its own: the
// card is already one, and an anchor inside an anchor is invalid markup. The
// corporate family graph further down the same page is the way there.
defineProps<{
  name: string
}>()
</script>

<template>
  <p class="text-default-500 mt-1 flex items-center gap-1 text-xs">
    <KunIcon name="lucide:corner-down-right" class="shrink-0" />
    <span class="truncate">经 {{ name }}</span>
  </p>
</template>
