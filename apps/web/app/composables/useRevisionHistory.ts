// Parameterised revision-history hook. Drives the GalgameRevisionList
// component for ALL five entity types (galgame + 4 taxonomy entities)
// so the UI logic lives in one place; entity-specific routing is the
// only differentiator and it folds into a tiny lookup table.
//
// Endpoints (kungal proxy paths — kungal's path_mapper handles the
// /galgame/ vs bare /tag namespace difference; see path_mapper.go):
//
//   galgame: GET /galgame/:gid/history/all    → list (kungal-mapped to camelCase)
//            GET /galgame/:gid/revisions/:rev/diff  → diff
//            POST /galgame/:gid/revert        → revert
//
//   taxonomy: GET /galgame-<entity>/:id/revisions[/:rev] → list / single
//             POST /galgame-<entity>/:id/revert
//
// Note the asymmetry: galgame already exposed a kungal-mapped list
// endpoint (history/all → revisions, camelCase via GetGalgameHistory)
// AND a server-side diff endpoint (/revisions/:rev/diff). The taxonomy
// path is ProxyGet for list/snapshot — there's no diff endpoint on
// wiki for taxonomies in U3, so the FE computes the diff client-side
// by pulling two adjacent snapshots (or current vs picked) and feeding
// them to GalgameSnapshotDiff. List/snapshot routes are camelCase for
// galgame (kungal handler) but snake_case verbatim for taxonomies
// (ProxyGet) — `Revision` is normalised to a shared shape for the UI.

// All four taxonomy entities (tag / official / engine / series) support
// revision history. Series surfaces only its own name/alias/description
// edits — membership changes (a galgame joining/leaving) are recorded as
// galgame-side revisions, not here — but that's still useful history, so
// series is included (an earlier version excluded it; that decision is
// removed).
export type RevisionEntity =
  | 'galgame'
  | 'tag'
  | 'official'
  | 'engine'
  | 'series'

export interface GalgameRevisionListItem {
  id: number
  revision: number
  action: string
  note: string
  user: KunUser
  is_minor: boolean
  created: Date | string
}

// Internal: raw wiki taxonomy revision row (snake_case ProxyGet
// passthrough). We normalise to GalgameRevisionListItem at the
// boundary so the UI sees one shape.
interface WikiTaxonomyRevisionRow {
  id: number
  revision: number
  action: string
  note?: string
  user_id?: number
  changed_fields?: string[]
  ref_count?: number
  created: string
}

interface EndpointSet {
  list: string
  rev: (n: number) => string
  diff: ((n: number) => string) | null // null = no server-side diff
  revert: string
}

const endpointsFor = (entity: RevisionEntity, id: number): EndpointSet => {
  if (entity === 'galgame') {
    return {
      list: `/galgame/${id}/history/all`,
      rev: (n) => `/galgame/${id}/revisions/${n}`,
      diff: (n) => `/galgame/${id}/revisions/${n}/diff`,
      revert: `/galgame/${id}/revert`
    }
  }
  // taxonomy: kungal's suffix-aware path_mapper takes care of the
  // /galgame/<entity>/ namespace rewrite.
  return {
    list: `/galgame-${entity}/${id}/revisions`,
    rev: (n) => `/galgame-${entity}/${id}/revisions/${n}`,
    diff: null,
    revert: `/galgame-${entity}/${id}/revert`
  }
}

export const useRevisionHistory = (
  entity: RevisionEntity,
  id: Ref<number> | ComputedRef<number> | number
) => {
  const idRef = computed(() => (typeof id === 'number' ? id : unref(id)))
  const ep = computed(() => endpointsFor(entity, idRef.value))

  const pageData = reactive({ page: 1, limit: 10 })

  // List response shape differs by entity (galgame: kungal-mapped
  // {items, total} camelCase; taxonomy: wiki verbatim — could be a
  // bare array or {items,total} depending on wiki). Try the kungal
  // {items,total} shape first; the data ref unwraps `items` when
  // present, falls back to the bare array otherwise.
  type RawListResponse =
    | { items: GalgameRevisionListItem[]; total: number }
    | WikiTaxonomyRevisionRow[]
    | null

  const { data, status, refresh } = useKunFetch<RawListResponse>(
    () => ep.value.list,
    {
      lazy: true,
      method: 'GET',
      query: pageData,
      watch: [() => ep.value.list, () => pageData.page]
    }
  )

  // Normalise: taxonomy rows arrive snake_case + no embedded user
  // object; the UI only needs revision / action / note / created /
  // is_minor / user.name|avatar. We fabricate a stub user from
  // user_id when no embedded user is present (UI shows just the id);
  // a later wiki upgrade can embed users and this code will pick it
  // up automatically via the GalgameRevisionListItem cast.
  const items = computed<GalgameRevisionListItem[]>(() => {
    const d = data.value
    if (!d) return []
    // Response may be a bare array (older wiki passthrough) or {items,total}
    // (galgame history + the hydrated taxonomy endpoint). Normalise EITHER
    // shape, and defensively fabricate a user stub for any row that lacks an
    // embedded `user` (a raw snake_case wiki row) — otherwise the list
    // renderer throws on `rev.user.name`.
    const rows = (Array.isArray(d) ? d : (d.items ?? [])) as Array<
      GalgameRevisionListItem & Partial<WikiTaxonomyRevisionRow>
    >
    return rows.map((r) => {
      if (r.user) {
        return r as GalgameRevisionListItem
      }
      return {
        id: r.id,
        revision: r.revision,
        action: r.action,
        note: r.note ?? '',
        is_minor: r.is_minor ?? false,
        created: r.created,
        user: {
          id: r.user_id ?? 0,
          name: r.user_id ? `#${r.user_id}` : '',
          avatar: ''
        } as KunUser
      }
    })
  })

  const total = computed(() => {
    const d = data.value
    if (!d) return 0
    if (Array.isArray(d)) return d.length
    return d.total ?? 0
  })

  // Diff: galgame has a server-side endpoint; taxonomy doesn't, so we
  // construct one from two snapshots (target vs current — which the
  // UI obtains by fetching the LATEST revision separately).
  //
  // `names` is the K-PR 2026-Q2 addition: a {id → displayName} dict
  // per related entity kind, scoped to ids referenced in this diff.
  // Optional — taxonomy diffs we synthesise locally don't have one,
  // and the renderer treats absence as "render raw ids".
  interface SnapshotDiff {
    changed_keys: Record<string, boolean>
    old: Record<string, unknown>
    new: Record<string, unknown>
    names?: {
      tags?: Record<string, string>
      officials?: Record<string, string>
      engines?: Record<string, string>
      series?: Record<string, string>
    }
  }

  const diffCache = reactive<Record<number, SnapshotDiff>>({})
  const diffLoading = ref<number | null>(null)

  const loadDiff = async (rev: number): Promise<SnapshotDiff | null> => {
    if (diffCache[rev]) return diffCache[rev]!
    diffLoading.value = rev
    try {
      if (ep.value.diff) {
        // galgame: server-side diff endpoint returns the canonical shape.
        const res = await kunFetch<SnapshotDiff>(ep.value.diff(rev), {
          method: 'GET'
        })
        if (res) diffCache[rev] = res
        return res ?? null
      }
      // taxonomy: no server-side diff endpoint, so synthesise "what THIS
      // revision changed" = this revision's snapshot vs the PREVIOUS
      // revision's snapshot (the next-older entry in the newest-first list).
      //
      // The previous code compared against the LATEST revision instead, so
      // clicking the newest (or the only) revision diffed a snapshot against
      // itself → always "无字段变化". The wiki already reports `changed_fields`
      // per revision, so we trust that for the changed-key set (it's the
      // authoritative "what changed here", and avoids flagging every field as
      // new on the first revision where there's no prior snapshot); we fall
      // back to a snapshot compare if it's absent.
      const target = await kunFetch<{
        snapshot?: Record<string, unknown>
        changed_fields?: string[]
      }>(ep.value.rev(rev), { method: 'GET' })
      if (!target) return null
      const newSnap = target.snapshot ?? {}

      const idx = items.value.findIndex((it) => it.revision === rev)
      const prevItem = idx >= 0 ? items.value[idx + 1] : undefined
      const prevRev = prevItem ? prevItem.revision : null
      let oldSnap: Record<string, unknown> = {}
      if (prevRev !== null) {
        const prev = await kunFetch<{ snapshot?: Record<string, unknown> }>(
          ep.value.rev(prevRev),
          { method: 'GET' }
        )
        oldSnap = prev?.snapshot ?? {}
      }

      const changed_keys: Record<string, boolean> = {}
      if (target.changed_fields?.length) {
        for (const k of target.changed_fields) changed_keys[k] = true
      } else {
        for (const k of new Set([
          ...Object.keys(oldSnap),
          ...Object.keys(newSnap)
        ])) {
          if (JSON.stringify(oldSnap[k]) !== JSON.stringify(newSnap[k])) {
            changed_keys[k] = true
          }
        }
      }
      const diff = { changed_keys, old: oldSnap, new: newSnap }
      diffCache[rev] = diff
      return diff
    } finally {
      diffLoading.value = null
    }
  }

  const reverting = ref<number | null>(null)

  const revert = async (rev: number): Promise<boolean> => {
    reverting.value = rev
    try {
      const res = await kunFetch(ep.value.revert, {
        method: 'POST',
        body: { revision: rev }
      })
      if (res !== null) {
        // Bust the local diff cache (latest may have shifted) and
        // refresh the list so the new 'reverted' revision shows up.
        // dynamic-delete is unavoidable here — diffCache is a reactive
        // proxy and reassigning would lose its identity; clearing keys
        // one by one is the supported way.
        for (const k of Object.keys(diffCache)) {
          // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
          delete diffCache[Number(k)]
        }
        await refresh()
        return true
      }
      return false
    } finally {
      reverting.value = null
    }
  }

  return {
    items,
    total,
    status,
    pageData,
    refresh,
    loadDiff,
    diffCache,
    diffLoading,
    revert,
    reverting
  }
}
