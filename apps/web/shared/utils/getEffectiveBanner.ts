interface BannerSource {
  effective_banner_url?: string
  banner?: string
}

type ImageVariant = 'mini' | '100' | '256'

const IMAGE_SERVICE_HASH_PATH = /\/[0-9a-f]{2}\/[0-9a-f]{2}\/[0-9a-f]+\.webp$/i

export const withImageVariant = (
  url: string,
  variant: ImageVariant
): string => {
  if (!url || !/\.webp$/i.test(url)) return url
  const sep = IMAGE_SERVICE_HASH_PATH.test(url) ? '_' : '-'
  return url.replace(/\.webp$/i, `${sep}${variant}.webp`)
}

export const withBannerVariant = (
  url: string,
  variant: Extract<ImageVariant, 'mini'>
): string => withImageVariant(url, variant)

export const getEffectiveBanner = (
  g?: BannerSource | null,
  opts?: { variant?: Extract<ImageVariant, 'mini'> }
): string => {
  if (!g) return ''
  const eff = g.effective_banner_url?.trim()
  const base = (eff || g.banner?.trim() || '').trim()
  if (!base || !opts?.variant) return base
  return withBannerVariant(base, opts.variant)
}

export const imageAspectRatio = (
  width?: number,
  height?: number,
  fallback = '16 / 9'
): string => (width && height ? `${width} / ${height}` : fallback)

export const resolveBannerThumbhash = (
  g?: {
    effective_banner_thumbhash?: string
    covers?: { sort_order: number; thumbhash?: string }[]
  } | null
): string => {
  if (!g) return ''
  if (g.effective_banner_thumbhash) return g.effective_banner_thumbhash
  const covers = g.covers
  if (!covers?.length) return ''
  const pinned = covers.find((c) => c.sort_order === 0) ?? covers[0]
  return pinned?.thumbhash ?? ''
}
