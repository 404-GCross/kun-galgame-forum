import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface RecentQuizGalgame {
  id: number
  name: string
  banner?: string
  thumbhash?: string
  officials?: string[]
}

const MAX_RECENT = 8

export const usePersistQuizGalgameStore = defineStore(
  'KUNGalgameQuizGalgame',
  () => {
    const recent = ref<RecentQuizGalgame[]>([])

    const add = (game: RecentQuizGalgame) => {
      if (!game?.id) {
        return
      }
      recent.value = [
        game,
        ...recent.value.filter((g) => g.id !== game.id)
      ].slice(0, MAX_RECENT)
    }

    const remove = (id: number) => {
      recent.value = recent.value.filter((g) => g.id !== id)
    }

    return { recent, add, remove }
  },
  { persist: { storage: piniaPluginPersistedstate.localStorage() } }
)
