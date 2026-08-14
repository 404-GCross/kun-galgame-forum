import {
  KUN_GALGAME_DIMENSIONS,
  type KunGalgameDim
} from '~/constants/galgame-rating'

export const ratingMean = (values: number[]) =>
  values.length ? values.reduce((sum, v) => sum + v, 0) / values.length : 0

export const ratingHistogram = (
  ratings: GalgameRatingCardOnGalgamePage[],
  max = 10
) => {
  const buckets = Array.from({ length: max }, () => 0)
  for (const rating of ratings) {
    const index = Math.round(rating.overall) - 1
    if (index >= 0 && index < max) {
      buckets[index]! += 1
    }
  }
  return buckets
}

export const ratingDimensionMeans = (
  ratings: GalgameRatingCardOnGalgamePage[]
) =>
  Object.fromEntries(
    KUN_GALGAME_DIMENSIONS.map((dim) => [
      dim,
      ratingMean(ratings.map((rating) => rating[dim]))
    ])
  ) as Record<KunGalgameDim, number>

export const ratingTally = (values: string[]) => {
  const tally = new Map<string, number>()
  for (const value of values) {
    tally.set(value, (tally.get(value) ?? 0) + 1)
  }
  return [...tally].sort((a, b) => b[1] - a[1])
}
