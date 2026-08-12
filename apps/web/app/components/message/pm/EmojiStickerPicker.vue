<script setup lang="ts">
import { emojiArray } from '~/constants/emoji'
import { stickerArray } from '~/constants/sticker'

const emit = defineEmits<{
  emoji: [emoji: string]
  sticker: [url: string]
}>()

const tab = ref<'emoji' | 'sticker'>('emoji')
</script>

<template>
  <div class="w-72 p-2 sm:w-80">
    <div class="bg-default-100 mb-2 flex rounded-full p-1 text-sm">
      <button
        type="button"
        @click="tab = 'emoji'"
        :class="
          cn(
            'flex-1 rounded-full py-1.5 transition-colors',
            tab === 'emoji'
              ? 'bg-background text-primary font-medium shadow-sm'
              : 'text-default-500 hover:text-default-700'
          )
        "
      >
        表情
      </button>
      <button
        type="button"
        @click="tab = 'sticker'"
        :class="
          cn(
            'flex-1 rounded-full py-1.5 transition-colors',
            tab === 'sticker'
              ? 'bg-background text-primary font-medium shadow-sm'
              : 'text-default-500 hover:text-default-700'
          )
        "
      >
        贴纸
      </button>
    </div>

    <KunOverlayScroll v-show="tab === 'emoji'" class="h-56">
      <div class="grid grid-cols-8 gap-0.5">
        <button
          v-for="(e, i) in emojiArray"
          :key="i"
          type="button"
          @click="emit('emoji', e)"
          class="hover:bg-default-100 flex aspect-square items-center justify-center rounded-md text-xl"
        >
          {{ e }}
        </button>
      </div>
    </KunOverlayScroll>

    <KunOverlayScroll v-show="tab === 'sticker'" class="h-56">
      <div class="grid grid-cols-4 gap-1">
        <button
          v-for="url in stickerArray"
          :key="url"
          type="button"
          @click="emit('sticker', url)"
          class="hover:bg-default-100 aspect-square rounded-md p-1"
        >
          <img
            :src="url"
            alt="sticker"
            loading="lazy"
            class="size-full object-contain"
          />
        </button>
      </div>
    </KunOverlayScroll>
  </div>
</template>
