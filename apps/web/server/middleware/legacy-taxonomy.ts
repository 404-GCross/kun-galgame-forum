import {
  parseWikiId,
  resolveLegacyTag,
  resolveLegacyEngine,
  resolveRenamedTaxonomyPath
} from '../utils/kunTaxonomyRedirects'
import { taxonomyDetailPath } from '../../shared/utils/kunTaxonomyPaths'

const LEGACY_RE = /^\/galgame-(tag|official|engine)\/(\d+)$/

export default defineEventHandler(async (event) => {
  if (event.method !== 'GET') return
  const [path, query] = event.path.split('?') as [string, string?]

  const withQuery = (to: string) => (query ? `${to}?${query}` : to)

  const renamed = resolveRenamedTaxonomyPath(path)
  if (renamed) return sendRedirect(event, withQuery(renamed), 301)

  const m = path.match(LEGACY_RE)
  if (!m) return

  const family = m[1]!
  const wikiId = parseWikiId(m[2])
  if (wikiId === null) return

  if (family === 'tag') {
    const res = resolveLegacyTag(wikiId)
    if (res.kind === 'redirect') return sendRedirect(event, res.to, 301)
    if (res.kind === 'gone') {
      throw createError({
        statusCode: 410,
        statusMessage: '该标签已在词表迁移中退役'
      })
    }
    return
  }

  if (family === 'engine') {
    const res = resolveLegacyEngine(wikiId)
    if (res.kind === 'redirect') return sendRedirect(event, res.to, 301)
    return
  }

  const id = await $fetch<{ data?: { id?: number } }>(
    `${useRuntimeConfig().apiBaseUrl}/api/galgame-official/legacy/${wikiId}`,
    { headers: { accept: 'application/json' } }
  )
    .then((r) => r?.data?.id)
    .catch(() => undefined)

  if (id) return sendRedirect(event, taxonomyDetailPath('official', id), 301)
})
