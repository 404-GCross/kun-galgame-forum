<script setup lang="ts">
import { useWindowScroll } from '@vueuse/core'

withDefaults(
  defineProps<{
    className?: string
  }>(),
  { className: '' }
)

const offsetClass = 'desktop-nav:left-[104px] desktop-nav:w-[calc(100%-116px)]'

const { y } = useWindowScroll()
const scrolled = computed(() => y.value > 8)
</script>

<template>
  <div
    :class="
      cn(
        'fixed top-0 z-30 mb-3 ml-0 shrink-0',
        'left-0 w-full',
        offsetClass,
        className
      )
    "
  >
    <div
      :class="
        cn(
          'mx-auto flex h-16 w-full max-w-7xl transform-gpu items-center justify-between rounded-b-lg border transition-[background-color,border-color,box-shadow,padding] duration-200',
          scrolled
            ? 'bg-content1 border-kun shadow-kun-sm px-3 backdrop-blur-md'
            : 'desktop-nav:px-0 border-transparent bg-transparent px-3 shadow-none'
        )
      "
    >
      <KunTopBarNav />
      <KunTopBarAvatar />
    </div>
  </div>
</template>
