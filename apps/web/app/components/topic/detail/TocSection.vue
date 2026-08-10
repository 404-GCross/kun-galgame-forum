<script setup lang="ts">
import type { TOCItem } from '~/composables/topic/useTopicTOC'
import { scrollToTOCElement } from '../_helper'

const props = defineProps<{
  items: TOCItem[]
  activeIds: string[]
}>()

const wrapper = ref<HTMLElement | null>(null)
const overlay = reactive({ top: 0, height: 0, visible: false })

const measure = () => {
  const w = wrapper.value
  if (!w) {
    overlay.visible = false
    return
  }
  const base = w.getBoundingClientRect().top
  let top = Infinity
  let bottom = -Infinity
  w.querySelectorAll<HTMLElement>('a[data-toc-id]').forEach((el) => {
    if (!props.activeIds.includes(el.dataset.tocId ?? '')) {
      return
    }
    const r = el.getBoundingClientRect()
    top = Math.min(top, r.top - base)
    bottom = Math.max(bottom, r.bottom - base)
  })

  if (top === Infinity) {
    overlay.visible = false
    return
  }
  overlay.top = top
  overlay.height = bottom - top
  overlay.visible = true
}

watch([() => props.activeIds, () => props.items], () => measure(), {
  flush: 'post'
})
const onResize = () => measure()
onMounted(() => {
  measure()
  window.addEventListener('resize', onResize, { passive: true })
})
onBeforeUnmount(() => window.removeEventListener('resize', onResize))
</script>

<template>
  <div ref="wrapper" class="relative">
    <div
      v-show="overlay.visible"
      class="bg-primary/10 pointer-events-none absolute inset-x-0 rounded-md transition-all duration-300 ease-out"
      :style="{ top: `${overlay.top}px`, height: `${overlay.height}px` }"
    />

    <ul class="relative z-10 space-y-1">
      <li
        v-for="item in items"
        :key="item.id"
        :style="{ paddingLeft: `${(item.level - 1) * 0.75}rem` }"
      >
        <a
          :data-toc-id="item.id"
          :href="`#${item.id}`"
          @click.prevent="scrollToTOCElement(item.id)"
          :class="
            cn(
              'block py-1 pr-2 pl-2.5 text-sm transition-colors duration-300',
              item.targeted
                ? 'text-primary font-bold'
                : activeIds.includes(item.id)
                  ? 'text-primary font-medium'
                  : 'text-default-500 hover:text-primary'
            )
          "
        >
          {{ item.text }}
        </a>
      </li>
    </ul>
  </div>
</template>
