// The 会社 detail fetch, shared by the two pages a maker's space is made of:
// the overview (/galgame/official/:id) and the games browser (…/game).
//
// Both need the same junk-id 404 and the same merged-id 301, and those are
// exactly the checks that rot once a second page copies them — the merged-id
// hop already cost one production 500 (see the note on the overview page), so
// it is written once and both pages get it identically.
//
// `subPath` keeps a reader on the page they asked for across that hop: a merged
// id hit at …/game lands on the survivor's …/game, not on its overview. The
// target is still built from the shared path builder, so it stays the FINAL
// form and the hop can never become a chain.
export const useGalgameOfficialDetail = async (
  query: Record<string, unknown>,
  subPath = ''
) => {
  const route = useRoute()
  const officialId = Number((route.params as { id: string }).id)

  // A junk segment (/galgame/official/null, crawler-made) becomes NaN and used
  // to ride all the way upstream, where the catalog answered 400 — pointless
  // round trips for a URL that can only ever be a 404. Answer it here.
  if (!Number.isInteger(officialId) || officialId <= 0) {
    throw createError({
      statusCode: 404,
      statusMessage: '未找到 Galgame 会社',
      fatal: true
    })
  }

  const { data, status } = await useKunFetch<GalgameOfficialDetail>(
    `/galgame-official/${officialId}`,
    { method: 'GET', query }
  )

  // A merged 会社 keeps its old id addressable, but only as a 301: the catalog
  // merges duplicate labels, and the loser's id has to land on the survivor's
  // page rather than render a copy of it. `moved_to` arrives instead of the
  // record, never alongside it, so nothing of the survivor is ever painted
  // under the old id.
  //
  // `navigateTo` is NOT an early return here, awaited or not: on the server it
  // only parks a redirect on ssrContext['~renderResponse'] and hands control
  // back, so the caller's setup AND its template still render — against the
  // tombstone payload (id 0, alias/galgame null). A throw from that render
  // preempts the parked redirect and the visitor gets a 500 instead of the 301
  // (prod: /galgame/official/13323 died on `alias.length`). So every page
  // template using this gates its root on `!data.moved_to` — reactive, so a
  // client-side hop stops painting the dead brand too.
  if (data.value?.moved_to) {
    await navigateTo(
      `${taxonomyDetailPath('official', data.value.moved_to)}${subPath}`,
      { redirectCode: 301, replace: true }
    )
  }

  // An unknown id is a real 404, not an empty 200 shell: this namespace went
  // live with no legacy id space behind it, so a miss means the entity does not
  // exist and a crawler must be told exactly that rather than indexing a blank
  // page.
  if (!data.value) {
    throw createError({
      statusCode: 404,
      statusMessage: '未找到 Galgame 会社',
      fatal: true
    })
  }

  return { officialId, data, status, moved: !!data.value.moved_to }
}
