// Legacy taxonomy URL shells (A2-3 / doc 106 R1).
//
// The taxonomy detail pages moved from the wiki's id space to the catalog's,
// and the live pages now live under a `/c/` segment:
//
//   /galgame-tag/{wikiId}       →  /galgame-tag/c/{catalogTagId}
//   /galgame-official/{wikiId}  →  /galgame-official/c/{catalogLabelId}
//   /galgame-engine/{wikiId}    →  /galgame-engine/c/{catalogEngineId}
//
// The split is not cosmetic. The two id spaces OVERLAP densely — 718 of the
// 1,530 mapped wiki tag ids are themselves live catalog tag ids, and 442 of the
// 1,507 retired ones are too — so a single path serving both meanings would
// silently render the wrong entity for whichever one lost. A distinct segment
// makes the question decidable forever: a bare number here is ALWAYS a wiki id,
// `/c/{n}` is ALWAYS a catalog id, no matter how either range grows later.
//
// This runs as global middleware rather than as shell PAGES so the redirect is
// issued before any rendering — a crawler gets a clean 301 with no HTML body,
// and there is no client-side hydration for a URL that is never meant to
// render. Paths that do not match the strict legacy shape fall straight
// through, including the `/c/…` pages themselves and the index pages.
//
// Three outcomes, and the difference between the last two is what a crawler
// acts on:
//   301  the entity has a canonical counterpart
//   410  it had none (the 1,507 retired tags) — permanently gone, so a crawler
//        drops it instead of re-checking forever
//   404  never a valid id in the first place
import {
  parseWikiId,
  resolveLegacyTag,
  resolveLegacyEngine
} from '../utils/kunTaxonomyRedirects'

// Strict shape: exactly one numeric segment under one of the three families.
// `/c/…` cannot match (it has a non-numeric segment), and neither can the
// index pages or any deeper path.
const LEGACY_RE = /^\/galgame-(tag|official|engine)\/(\d+)$/

export default defineEventHandler(async (event) => {
  if (event.method !== 'GET') return
  const path = event.path.split('?')[0]!
  const m = path.match(LEGACY_RE)
  if (!m) return

  const family = m[1]!
  const wikiId = parseWikiId(m[2])
  if (wikiId === null) return // not a usable id → let the router 404 it

  if (family === 'tag') {
    const res = resolveLegacyTag(wikiId)
    if (res.kind === 'redirect') return sendRedirect(event, res.to, 301)
    if (res.kind === 'gone') {
      throw createError({
        statusCode: 410,
        statusMessage: '该标签已在词表迁移中退役'
      })
    }
    return // unknown wiki id → fall through to the router's 404
  }

  if (family === 'engine') {
    // NOT the identity mapping: 52 of the 189 engines landed on a different
    // catalog id, so this has to be looked up rather than assumed.
    const res = resolveLegacyEngine(wikiId)
    if (res.kind === 'redirect') return sendRedirect(event, res.to, 301)
    return
  }

  // Makers resolve at runtime through the registry's external-ref index (A2-0
  // registered 100% of them), so future merges keep redirecting correctly —
  // something a frozen map could not do. An unreachable API falls through to a
  // 404 rather than erroring the request: a missing redirect is recoverable,
  // a 500 on a crawled URL is not.
  const id = await $fetch<{ data?: { id?: number } }>(
    `${useRuntimeConfig().apiBaseUrl}/api/galgame-official/legacy/${wikiId}`,
    { headers: { accept: 'application/json' } }
  )
    .then((r) => r?.data?.id)
    .catch(() => undefined)

  if (id) return sendRedirect(event, `/galgame-official/c/${id}`, 301)
})
