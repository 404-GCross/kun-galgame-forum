// Legacy taxonomy URL → new catalog-id URL (A2-3 / doc 106 R1).
//
// The taxonomy detail pages moved from the wiki's id space to the catalog's.
// The two spaces OVERLAP densely — 718 of the 1,530 mapped wiki tag ids are
// themselves live catalog tag ids, and 442 of the 1,507 retired ones are too —
// so a bare `/galgame-tag/{n}` cannot be resolved to one meaning or the other.
// That is why the new pages live under a distinct `/c/` segment and these
// legacy paths became pure redirect shells: a bare numeric segment is ALWAYS a
// wiki id, `/c/{n}` is ALWAYS a catalog id, and neither can be mistaken for the
// other no matter how the two id ranges grow.
//
// The maps are frozen exports of the production registry (infra
// refs/proj/132-artifacts + 127-artifacts) and are vendored rather than fetched:
// they describe a one-time historical migration, so they can never drift, and a
// redirect must not depend on a live service being reachable.

import tagRedirects from '../data/wiki-tag-redirects.json'
import engineRedirects from '../data/wiki-engine-redirects.json'
import tagGone from '../data/wiki-tag-gone.json'

const tagMap = tagRedirects as Record<string, number>
const engineMap = engineRedirects as Record<string, number>
// The wiki tags with no canonical counterpart. They are 410 rather than 404:
// the URL was valid and the resource is permanently gone, which is what tells a
// crawler to drop it instead of re-checking forever.
const goneTags = new Set<number>(tagGone as number[])

export type LegacyResolution =
  | { kind: 'redirect'; to: string }
  | { kind: 'gone' }
  | { kind: 'missing' }

/** Parse a legacy path segment as a positive wiki id. */
export const parseWikiId = (raw: string | undefined): number | null => {
  if (!raw) return null
  const n = Number(raw)
  return Number.isInteger(n) && n > 0 ? n : null
}

/** Resolve a legacy tag id: 301 when mapped, 410 when retired, else 404. */
export const resolveLegacyTag = (wikiId: number): LegacyResolution => {
  const target = tagMap[String(wikiId)]
  if (target) return { kind: 'redirect', to: `/galgame-tag/c/${target}` }
  if (goneTags.has(wikiId)) return { kind: 'gone' }
  return { kind: 'missing' }
}

/**
 * Resolve a legacy engine id. The mapping is NOT the identity — 52 of the 189
 * engines landed on a different catalog id — so it has to be looked up rather
 * than assumed.
 */
export const resolveLegacyEngine = (wikiId: number): LegacyResolution => {
  const target = engineMap[String(wikiId)]
  if (target) return { kind: 'redirect', to: `/galgame-engine/c/${target}` }
  return { kind: 'missing' }
}
