export const useTabQuery = (defaultValue: string, key = 'tab') => {
  const route = useRoute()
  const router = useRouter()

  return computed<string>({
    get() {
      const v = route.query[key]
      const val = Array.isArray(v) ? v[0] : v
      return val || defaultValue
    },
    set(val) {
      const query = { ...route.query }
      if (val === defaultValue) {
        delete query[key]
      } else {
        query[key] = val
      }
      router.replace({ query })
    }
  })
}
