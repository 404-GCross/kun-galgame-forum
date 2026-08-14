import { describe, expect, it } from 'vitest'
import {
  GALGAME_RATING_TIER_CONST,
  galgameRatingTier,
  type GalgameRatingTierRule
} from './galgameRatingTier'

const vndb: GalgameRatingTierRule = {
  cutoffs: [5.5, 6.5, 7.4, 8.0],
  minVotes: 10
}
const erogamescape: GalgameRatingTierRule = {
  cutoffs: [60, 70, 80, 85],
  minVotes: 10
}
const local: GalgameRatingTierRule = {
  cutoffs: [6.8, 7.3, 7.9, 8.3],
  minVotes: 3
}

describe('galgameRatingTier', () => {
  it('places a score in every tier', () => {
    expect(galgameRatingTier(vndb, 8.4, 100)).toBe('god')
    expect(galgameRatingTier(vndb, 7.6, 100)).toBe('masterpiece')
    expect(galgameRatingTier(vndb, 6.9, 100)).toBe('good')
    expect(galgameRatingTier(vndb, 6.0, 100)).toBe('average')
    expect(galgameRatingTier(vndb, 4.2, 100)).toBe('bad')
  })

  it('treats every cutoff as left-closed', () => {
    expect(galgameRatingTier(vndb, 8.0, 100)).toBe('god')
    expect(galgameRatingTier(vndb, 7.99, 100)).toBe('masterpiece')
    expect(galgameRatingTier(vndb, 7.4, 100)).toBe('masterpiece')
    expect(galgameRatingTier(vndb, 6.5, 100)).toBe('good')
    expect(galgameRatingTier(vndb, 5.5, 100)).toBe('average')
    expect(galgameRatingTier(vndb, 5.49, 100)).toBe('bad')
  })

  it('reads cutoffs on the source scale, not a normalized one', () => {
    expect(galgameRatingTier(erogamescape, 85, 100)).toBe('god')
    expect(galgameRatingTier(erogamescape, 8.5, 100)).toBe('bad')
  })

  it('withholds a tier below the vote gate', () => {
    expect(galgameRatingTier(vndb, 9.5, 9)).toBeNull()
    expect(galgameRatingTier(vndb, 9.5, 10)).toBe('god')
    expect(galgameRatingTier(local, 9.5, 2)).toBeNull()
    expect(galgameRatingTier(local, 9.5, 3)).toBe('god')
  })

  it('withholds a tier when there is no score at all', () => {
    expect(galgameRatingTier(vndb, null, 100)).toBeNull()
    expect(galgameRatingTier(vndb, undefined, 100)).toBeNull()
    expect(galgameRatingTier(vndb, 8.4, 0)).toBeNull()
    expect(galgameRatingTier(vndb, 8.4, null)).toBeNull()
  })

  it('keeps the local cutoffs inside the range the bayesian prior can reach', () => {
    // The local score is not a raw mean: list_repo.go blends it with a prior of
    // C=10 votes at the site mean (~7.42), which squeezes small samples toward
    // the middle. These cutoffs are picked for that squeezed range: three
    // perfect votes reach 名作 but not 神作, and the prior — not minVotes — is
    // what a raw mean would need a far larger gate to achieve.
    const bayesian = (votes: number[]) =>
      (10 * 7.42 + votes.reduce((sum, v) => sum + v, 0)) / (10 + votes.length)
    expect(bayesian([10])).toBeCloseTo(7.65, 2)
    expect(galgameRatingTier(local, bayesian([10]), 1)).toBeNull()
    expect(galgameRatingTier(local, bayesian([10, 10, 10]), 3)).toBe(
      'masterpiece'
    )
    expect(galgameRatingTier(local, bayesian(Array(10).fill(10)), 10)).toBe(
      'god'
    )
    expect(galgameRatingTier(local, bayesian(Array(10).fill(1)), 10)).toBe(
      'bad'
    )
  })

  it('orders the tier vocabulary from worst to best', () => {
    expect(GALGAME_RATING_TIER_CONST).toEqual([
      'bad',
      'average',
      'good',
      'masterpiece',
      'god'
    ])
  })
})
