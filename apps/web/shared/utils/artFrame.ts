// Framing for artwork whose shape is only known at runtime — the catalog's
// character busts and 全身立绘.
//
// The rule these helpers encode: KNOW the ratio and the picture keeps its own
// shape, contained, with nothing cropped and no layout jump. Do NOT know it and
// impose NO frame at all — the browser then lays the picture out at its own
// intrinsic proportions once it lands.
//
// That second half was originally a guessed fallback frame (3:4 for a bust,
// square for a 立绘) and it was simply wrong: those numbers describe the CROPPED
// thumbnail variants, while a modal and a page header show the ORIGINAL. A bust
// cover-cropped into a guessed 3:4 is a picture of someone's chin. Trading a
// little layout shift for a correct shape is the right way round — the shift
// only happens when image_service could not answer at all.

import type { GalgameArtMeta } from '../types/galgame'

/**
 * The frame for one artwork: what box to reserve and how to sit in it.
 * An empty `aspectRatio` means "reserve nothing" — KunImage then renders the
 * picture at its natural size instead of fitting it into a box.
 */
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
 * its real ratio, contained — and when the shape is unknown, no frame, so the
 * picture arrives in its own proportions rather than a plausible-looking crop.
 *
 * Pass the FIRST meta that is actually about the picture being rendered; the
 * later arguments are consulted only while the earlier ones are absent (a
 * roster line and the character's own record describe the same artwork, and
 * whichever answered first is equally true).
 */
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
