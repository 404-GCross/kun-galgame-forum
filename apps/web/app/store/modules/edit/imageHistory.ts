import { defineStore } from 'pinia'
import { ref } from 'vue'

const MAX_IMAGE_HISTORY = 24

export const usePersistImageHistoryStore = defineStore(
  'KUNGalgameImageHistory',
  () => {
    const images = ref<string[]>([])

    const add = (src: string) => {
      if (!src) {
        return
      }
      images.value = [src, ...images.value.filter((s) => s !== src)].slice(
        0,
        MAX_IMAGE_HISTORY
      )
    }

    const remove = (src: string) => {
      images.value = images.value.filter((s) => s !== src)
    }

    return { images, add, remove }
  },
  { persist: { storage: piniaPluginPersistedstate.localStorage() } }
)
