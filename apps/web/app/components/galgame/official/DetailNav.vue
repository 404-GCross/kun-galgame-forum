<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

// A maker's space is two pages rather than one long scroll: 会社资料 (identity
// and the corporate family) and 作品 (the filterable catalogue). The split
// exists because a publisher with a deep family tree drew a tall, mostly empty
// tree between the header and the games — so the thing the page is FOR started
// below the fold, with the filter bar stranded under it.
//
// Route-backed tabs (KunTabItem.href), not local state: each half is its own
// URL, so it is shareable, indexable, and survives a refresh. `model-value` is
// bound one-way off the current path — navigation moves the highlight, not the
// other way round.
const props = defineProps<{
  officialId: number
  galgameCount?: number
}>()

const route = useRoute()
const basePath = computed(() =>
  taxonomyDetailPath('official', props.officialId)
)

const items = computed<KunTabItem[]>(() => [
  {
    value: 'profile',
    textValue: '会社资料',
    icon: 'lucide:building-2',
    href: basePath.value
  },
  {
    value: 'game',
    // The count belongs on the tab: it is the one number that tells a reader
    // whether the other half is worth opening.
    textValue: props.galgameCount ? `作品 ${props.galgameCount}` : '作品',
    icon: 'lucide:gamepad-2',
    href: `${basePath.value}/game`
  }
])

const activeTab = computed(() =>
  route.path.startsWith(`${basePath.value}/game`) ? 'game' : 'profile'
)
</script>

<template>
  <KunTab :items="items" :model-value="activeTab" variant="light" size="sm" />
</template>
