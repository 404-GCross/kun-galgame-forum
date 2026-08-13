<script setup lang="ts">
import { getLoli } from './getLoli'

const FRAME_W = 367
const FRAME_H = 602
const CANVAS_W = 504
const CANVAS_H = 925
const SCALE = 0.7
const stageW = Math.round(FRAME_W * SCALE)
const stageH = Math.round(FRAME_H * SCALE)

const loliData = ref({
  loliBodyLeft: '',
  loliBodyTop: '',
  loliEyeLeft: '',
  loliEyeTop: '',
  loliBrowLeft: '',
  loliBrowTop: '',
  loliMouthLeft: '',
  loliMouthTop: '',
  loliFaceLeft: '',
  loliFaceTop: '',
  body: '',
  eye: '',
  brow: '',
  mouth: '',
  face: '',
  bbox: { left: 137, top: 323, width: 367, height: 602 }
})

const canvasStyle = computed(() => {
  const b = loliData.value.bbox
  const x0 = b.left + b.width / 2 - FRAME_W / 2
  const y0 = b.top + b.height / 2 - FRAME_H / 2
  return {
    width: `${CANVAS_W}px`,
    height: `${CANVAS_H}px`,
    transform: `scale(${SCALE}) translate(${-x0}px, ${-y0}px)`,
    transformOrigin: 'top left'
  }
})

const ready = ref(false)

const decode = (src: string) => {
  if (!src) return Promise.resolve()
  const img = new Image()
  img.src = src
  return img.decode().catch(() => {})
}

const reroll = async () => {
  const data = await getLoli()
  await Promise.all(
    [data.body, data.eye, data.brow, data.mouth, data.face].map(decode)
  )
  loliData.value = data
  ready.value = true
}

onMounted(reroll)
</script>

<template>
  <div
    class="hidden shrink-0 sm:block"
    :style="{ width: `${stageW}px`, height: `${stageH}px` }"
  >
    <KunTooltip
      v-if="ready && loliData.body"
      text="点击换一个孩子"
      position="left"
    >
      <div
        class="relative cursor-pointer overflow-hidden"
        :style="{ width: `${stageW}px`, height: `${stageH}px` }"
        @click="reroll"
      >
        <div class="absolute top-0 left-0" :style="canvasStyle">
          <img
            class="absolute max-w-none"
            :src="loliData.body"
            alt="ren"
            :style="{ left: loliData.loliBodyLeft, top: loliData.loliBodyTop }"
          />
          <img
            class="absolute max-w-none"
            :src="loliData.eye"
            alt="ren"
            :style="{ left: loliData.loliEyeLeft, top: loliData.loliEyeTop }"
          />
          <img
            class="absolute max-w-none"
            :src="loliData.brow"
            alt="ren"
            :style="{ left: loliData.loliBrowLeft, top: loliData.loliBrowTop }"
          />
          <img
            class="absolute max-w-none"
            :src="loliData.mouth"
            alt="ren"
            :style="{
              left: loliData.loliMouthLeft,
              top: loliData.loliMouthTop
            }"
          />
          <img
            class="absolute max-w-none"
            :src="loliData.face"
            alt="ren"
            :style="{ left: loliData.loliFaceLeft, top: loliData.loliFaceTop }"
          />
        </div>
      </div>
    </KunTooltip>
  </div>
</template>
