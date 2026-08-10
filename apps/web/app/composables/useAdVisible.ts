import { storeToRefs } from 'pinia'

export const useAdVisible = () => {
  const { roles } = storeToRefs(usePersistUserStore())
  return computed(() => roles.value.length === 0)
}
