// Framing for artwork whose shape is only known at runtime — the catalog's
// character busts and 全身立绘.
//
// The rule these helpers encode: KNOW the ratio and the picture keeps its own
// shape, contained, with nothing cropped and no layout jump. Do NOT know it and
// the caller's fallback frame applies, which is the pre-existing behaviour
// rather than a broken one. A guessed ratio is never good enough here, because
// standing art is square for some titles and distinctly tall for others.

import type { GalgameArtMeta } from '../types/galgame'

/** The frame for one artwork: what box to reserve and how to sit in it. */
export interface KunArtFrame {
  aspectRatio: string
  objectFit: 'cover' | 'contain'
  thumbhash?: string
}

const ratioOf = (meta?: GalgameArtMeta): string =>
  meta && meta.width > 0 && meta.height > 0
    ? `${meta.width}/${meta.height}`
    : ''

/**
 * The frame for a single artwork standing on its own (a modal, a page header):
 * its real ratio, contained. `fallback` covers the unresolved case — pass the
 * frame that was right before dimensions existed.
 */
export const artFrame = (
  meta: GalgameArtMeta | undefined,
  fallback: KunArtFrame
): KunArtFrame => {
  const ratio = ratioOf(meta)
  return ratio
    ? { aspectRatio: ratio, objectFit: 'contain', thumbhash: meta?.thumbhash }
    : { ...fallback, thumbhash: meta?.thumbhash }
}

/**
 * ONE ratio for a GRID of artworks: the median of the shapes actually present.
 *
 * Per-item ratios are right for a lone picture and wrong for a grid — CSS grid
 * rows are as tall as their tallest cell, so mixed ratios leave every other
 * card sitting in a pool of dead space with its caption pushed out of line. The
 * median instead fits the set: a game whose 立绘 are all square gets square
 * frames, one whose art is tall gets tall frames, and the occasional odd one
 * out is contained inside with a thin band rather than cropped.
 *
 * `fallback` applies when nothing in the set resolved.
 */
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
