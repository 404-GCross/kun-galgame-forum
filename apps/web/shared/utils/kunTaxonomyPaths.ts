export const TAXONOMY_FAMILIES = ['tag', 'official', 'engine'] as const
export type TaxonomyFamily = (typeof TAXONOMY_FAMILIES)[number]

export const taxonomyIndexPath = (family: TaxonomyFamily) =>
  `/galgame/${family}`

export const taxonomyDetailPath = (family: TaxonomyFamily, catalogId: number) =>
  `/galgame/${family}/${catalogId}`
