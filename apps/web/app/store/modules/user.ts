import { defineStore } from 'pinia'
import { ref } from 'vue'
import { withImageVariant } from '../../../shared/utils/getEffectiveBanner'
import type { UserStore } from '../types/user'

export const usePersistUserStore = defineStore(
  'KUNGalgameUser',
  () => {
    const id = ref<UserStore['id']>(0)
    const sub = ref<UserStore['sub']>('')
    const name = ref<UserStore['name']>('')
    const avatar = ref<UserStore['avatar']>('')
    const avatarMin = ref<UserStore['avatarMin']>('')
    const moemoepoint = ref<UserStore['moemoepoint']>(0)
    const roles = ref<UserStore['roles']>([])
    const isCheckIn = ref<UserStore['isCheckIn']>(false)
    const dailyToolsetUploadBytes = ref<UserStore['dailyToolsetUploadBytes']>(0)

    const setUserInfo = (user: UserStore) => {
      id.value = user.id
      sub.value = user.sub
      name.value = user.name
      avatar.value = user.avatar
      avatarMin.value = withImageVariant(user.avatar, '100')
      moemoepoint.value = user.moemoepoint
      roles.value = user.roles
      isCheckIn.value = user.isCheckIn
      dailyToolsetUploadBytes.value = user.dailyToolsetUploadBytes
    }

    const setProfileInfo = (profile: {
      name: string
      avatar: string
      roles: string[]
    }) => {
      name.value = profile.name
      avatar.value = profile.avatar
      avatarMin.value = profile.avatar
        ? withImageVariant(profile.avatar, '100')
        : ''
      roles.value = profile.roles
    }

    const resetUser = () => {
      id.value = 0
      sub.value = ''
      name.value = ''
      avatar.value = ''
      avatarMin.value = ''
      moemoepoint.value = 0
      roles.value = []
      isCheckIn.value = false
      dailyToolsetUploadBytes.value = 0
    }

    return {
      id,
      sub,
      name,
      avatar,
      avatarMin,
      moemoepoint,
      roles,
      isCheckIn,
      dailyToolsetUploadBytes,
      setUserInfo,
      setProfileInfo,
      resetUser
    }
  },
  {
    persist: true
  }
)
