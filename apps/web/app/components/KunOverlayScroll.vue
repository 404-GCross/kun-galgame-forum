<script setup lang="ts">
import {
  OverlayScrollbarsComponent,
  type OverlayScrollbarsComponentRef
} from 'overlayscrollbars-vue'
import type { PartialOptions } from 'overlayscrollbars'

const props = withDefaults(
  defineProps<{
    defer?: boolean
  }>(),
  { defer: true }
)

const colorMode = useColorMode()
const options = computed<PartialOptions>(() => ({
  scrollbars: {
    theme: colorMode.value === 'dark' ? 'os-theme-light' : 'os-theme-dark',
    autoHide: 'leave',
    autoHideDelay: 500
  }
}))

const osRef = ref<OverlayScrollbarsComponentRef | null>(null)

const getViewport = (): HTMLElement | null =>
  osRef.value?.osInstance()?.elements().viewport ?? null

defineExpose({ getViewport })
</script>

<template>
  <OverlayScrollbarsComponent ref="osRef" :options="options" :defer="props.defer">
    <slot />
  </OverlayScrollbarsComponent>
</template>
