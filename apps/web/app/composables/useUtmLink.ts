export const useUtmLink = () => {
  const host = useRequestURL().host

  return (rawUrl: string): string => {
    if (!rawUrl) {
      return rawUrl
    }
    try {
      const url = new URL(
        /^https?:\/\//i.test(rawUrl) ? rawUrl : `https://${rawUrl}`
      )
      url.searchParams.set('utm_source', host)
      return url.toString()
    } catch {
      return rawUrl
    }
  }
}
