<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { kunUpdateLogTabItem } from '~/constants/update'

const activeTab = computed(
  () => useRoute().fullPath.split('/').pop() || 'history'
)

const isMobile = useMediaQuery('(max-width: 640px)')
const tabOrientation = computed(() =>
  isMobile.value ? 'horizontal' : 'vertical'
)
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start">
      <div class="shrink-0 sm:w-32">
        <KunTab
          :items="kunUpdateLogTabItem"
          :model-value="activeTab"
          :orientation="tabOrientation"
          variant="underlined"
          color="primary"
          full-width
        />
      </div>

      <div class="min-w-0 flex-1">
        <NuxtPage />
      </div>
    </div>
  </div>
</template>
