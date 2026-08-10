<script setup lang="ts">
import { kunSubSites } from '~/constants/layout'
</script>

<template>
  <footer class="text-default-500 space-y-2.5 text-center text-xs">

    <div class="grid grid-cols-3 gap-x-2 gap-y-1">
      <div
        v-for="(site, index) in kunSubSites"
        :key="index"
        class="text-center"
      >
        <KunLink size="sm" :to="site.link" target="_blank">
          {{ site.short }}
        </KunLink>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-center gap-1.5">
      <a
        :href="kungal.domain.main"
        target="_blank"
        class="flex items-center"
        :aria-label="kungal.titleShort"
      >
        <KunImage class="size-5 rounded-md" src="/favicon.webp" :alt="kungal.titleShort" />
      </a>
      <span>© 2023 ~ Forever! {{ kungal.titleShort }}</span>
    </div>

    <div class="flex flex-wrap items-center justify-center gap-1">
      <KunLink
        :to="kungal.github"
        size="sm"
        target="_blank"
        rel="noopener noreferrer"
      >
        GitHub
      </KunLink>
      <span>开源</span>
    </div>
  </footer>
</template>
