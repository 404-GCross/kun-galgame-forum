import { taxonomyDetailPath } from '../../shared/utils/kunTaxonomyPaths'
import tagRedirects from '../data/wiki-tag-redirects.json'
import engineRedirects from '../data/wiki-engine-redirects.json'
import tagGone from '../data/wiki-tag-gone.json'

const tagMap = tagRedirects as Record<string, number>
const engineMap = engineRedirects as Record<string, number>
const goneTags = new Set<number>(tagGone as number[])

export type LegacyResolution =
  | { kind: 'redirect'; to: string }
  | { kind: 'gone' }
  | { kind: 'missing' }

export const parseWikiId = (raw: string | undefined): number | null => {
  if (!raw) return null
  const n = Number(raw)
  return Number.isInteger(n) && n > 0 ? n : null
}

export const resolveLegacyTag = (wikiId: number): LegacyResolution => {
  const target = tagMap[String(wikiId)]
  if (target) return { kind: 'redirect', to: taxonomyDetailPath('tag', target) }
  if (goneTags.has(wikiId)) return { kind: 'gone' }
  return { kind: 'missing' }
}

export const resolveLegacyEngine = (wikiId: number): LegacyResolution => {
  const target = engineMap[String(wikiId)]
  if (target)
    return { kind: 'redirect', to: taxonomyDetailPath('engine', target) }
  return { kind: 'missing' }
}

const RENAMED_INDEX_RE = /^\/galgame-(tag|official|engine)$/
const RENAMED_DETAIL_RE = /^\/galgame-(tag|official|engine)\/c\/(\d+)$/

export const resolveRenamedTaxonomyPath = (path: string): string | null => {
  const index = path.match(RENAMED_INDEX_RE)
  if (index) return taxonomyIndexPath(index[1] as TaxonomyFamily)

  const detail = path.match(RENAMED_DETAIL_RE)
  if (detail) {
    const catalogId = Number(detail[2])
    if (!Number.isInteger(catalogId) || catalogId <= 0) return null
    return taxonomyDetailPath(detail[1] as TaxonomyFamily, catalogId)
  }

  return null
}
