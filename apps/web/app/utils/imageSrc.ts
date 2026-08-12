export const imageHashUrl = (
  base: string,
  hash: string,
  variant?: string
): string => {
  const b = (base || '').replace(/\/+$/, '')
  const file = variant ? `${hash}_${variant}.webp` : `${hash}.webp`
  return `${b}/${hash.slice(0, 2)}/${hash.slice(2, 4)}/${file}`
}

export const imageCdnBase = (): string => {
  try {
    const v = useRuntimeConfig().public.imageCdnBase as string | undefined
    if (v) return v.replace(/\/+$/, '')
  } catch {
    // useRuntimeConfig throws outside a Nuxt context (plain unit tests, module
    // scope). Fall through to the hardcoded default.
  }
  return 'https://image.kungal.iloveren.link'
}

export const imageTokenUrl = (tokenOrHash: string): string => {
  if (!tokenOrHash || tokenOrHash.startsWith('http')) return tokenOrHash
  const hash = tokenOrHash.startsWith('/image/')
    ? tokenOrHash.slice('/image/'.length)
    : tokenOrHash
  return imageHashUrl(imageCdnBase(), hash)
}
