export const useAuthModal = () => {
  const isOpen = useState('kun-auth-modal-open', () => false)

  return {
    isOpen,
    open: () => {
      isOpen.value = true
    },
    close: () => {
      isOpen.value = false
    }
  }
}

export const requireLogin = (): boolean => {
  if (usePersistUserStore().id) {
    return true
  }
  useAuthModal().open()
  return false
}
