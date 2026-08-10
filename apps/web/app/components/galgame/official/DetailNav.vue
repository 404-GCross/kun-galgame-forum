<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

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
