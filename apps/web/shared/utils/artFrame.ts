import type { GalgameArtMeta } from '../types/galgame'

export interface KunArtFrame {
  aspectRatio: string
  objectFit: 'cover' | 'contain'
  thumbhash?: string
}

const ratioOf = (meta?: GalgameArtMeta): string =>
  meta && meta.width > 0 && meta.height > 0
    ? `${meta.width}/${meta.height}`
    : ''

export const artFrame = (
  ...candidates: (GalgameArtMeta | undefined)[]
): KunArtFrame => {
  const meta = candidates.find((m) => !!ratioOf(m))
  return {
    aspectRatio: ratioOf(meta),
    objectFit: 'contain',
    thumbhash: candidates.find((m) => m?.thumbhash)?.thumbhash
  }
}

export const artGridRatio = (
  metas: (GalgameArtMeta | undefined)[],
  fallback: string
): string => {
  const ratios = metas
    .filter((m): m is GalgameArtMeta => !!m && m.width > 0 && m.height > 0)
    .map((m) => m.width / m.height)
    .sort((a, b) => a - b)
  if (!ratios.length) {
    return fallback
  }
  const median = ratios[Math.floor(ratios.length / 2)]!
  return `${median.toFixed(4)}/1`
}
