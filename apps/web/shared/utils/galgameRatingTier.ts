export const GALGAME_RATING_TIER_CONST = [
  'bad',
  'average',
  'good',
  'masterpiece',
  'god'
] as const
export type GalgameRatingTierKey = (typeof GALGAME_RATING_TIER_CONST)[number]

export interface GalgameRatingTierRule {
  // The four boundaries between the five tiers, ascending, expressed on the
  // source's OWN scale — erogamescape's are out of 100, dlsite's out of 5.
  // Never compare two sources by these numbers; compare them by the tier.
  cutoffs: readonly [number, number, number, number]
  minVotes: number
}

export const galgameRatingTier = (
  rule: GalgameRatingTierRule,
  score: number | null | undefined,
  voteCount: number | null | undefined
): GalgameRatingTierKey | null => {
  if (score == null || voteCount == null || voteCount < rule.minVotes) {
    return null
  }
  const index = rule.cutoffs.filter((cutoff) => score >= cutoff).length
  return GALGAME_RATING_TIER_CONST[index] ?? null
}
