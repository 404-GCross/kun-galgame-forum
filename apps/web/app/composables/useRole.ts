import { storeToRefs } from 'pinia'

export const ROLE = {
  creator: 'creator',
  moderator: 'moderator',
  admin: 'admin',
  ren: 'ren'
} as const

export const useRole = () => {
  const { roles } = storeToRefs(usePersistUserStore())

  const canModerate = computed(() =>
    roles.value.some(
      (r) => r === ROLE.moderator || r === ROLE.admin || r === ROLE.ren
    )
  )
  const canAdminister = computed(() =>
    roles.value.some((r) => r === ROLE.admin || r === ROLE.ren)
  )
  const isCreator = computed(() => roles.value.includes(ROLE.creator))

  return { canModerate, canAdminister, isCreator }
}
