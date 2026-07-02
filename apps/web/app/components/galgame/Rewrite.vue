<script setup lang="ts">
defineProps<{
  galgame: GalgameDetail
}>()

const { galgamePR } = storeToRefs(useTempGalgamePRStore())
const { id: currentUserId } = usePersistUserStore()
const { canModerate } = useRole()

const isOpening = ref(false)

const handleRewriteGalgame = async (galgame: GalgameDetail) => {
  if (isOpening.value) return
  isOpening.value = true

  // Wiki PR semantics are REPLACE-ALL for aliases / tag_ids /
  // official_ids / engine_ids / links: whatever we submit becomes the
  // entire set. So the working copy must be hydrated COMPLETELY from
  // current data, or a save silently wipes anything we left out.
  //
  // tag/official/engine come embedded in the detail payload already;
  // links do NOT — fetch them explicitly. If that fetch fails we keep
  // links empty but loudly warn, so the user can re-confirm the links
  // section instead of unknowingly clearing every link on submit.
  const linkRes = await kunFetch<GalgameLink[]>(
    `/galgame/${galgame.id}/link/all`,
    { method: 'GET' }
  )
  if (linkRes === null) {
    useMessage(
      '未能加载该 Galgame 当前的相关链接, 提交前请在「相关链接」区核对, 否则可能清空已有链接',
      'warn',
      5000
    )
  }

  galgamePR.value[0] = {
    id: galgame.id,
    vndb_id: galgame.vndb_id,
    name: galgame.name,
    // Editor (type="rewrite") edits raw markdown — use `markdown`, NOT
    // `introduction` (that field is pre-rendered HTML).
    introduction: galgame.markdown,
    content_limit: galgame.content_limit as 'sfw' | 'nsfw',
    age_limit: galgame.age_limit,
    original_language: galgame.original_language as Language,
    alias: [...galgame.alias],
    tags: [...galgame.tag],
    officials: [...galgame.official],
    engines: [...galgame.engine],
    links: (linkRes ?? []).map((l) => ({ name: l.name, link: l.link })),
    // Baseline in the SAME wire shape Footer compares (trimmed, blank-filtered),
    // so an untouched links section round-trips as "unchanged" → omitted →
    // preserved. A FAILED link fetch leaves both empty → unchanged → preserved
    // (not an empty replace-all that wipes every existing link on merge).
    links_baseline: JSON.stringify(
      (linkRes ?? [])
        .map((l) => ({ name: l.name.trim(), link: l.link.trim() }))
        .filter((l) => l.name.length > 0 && l.link.length > 0)
    ),
    note: '',
    // U1: hydrate from detail; nil → empty (treated as "unknown" by the
    // schema's `"" | YYYY-MM-DD` refinement). The wire ships ISO
    // datetimes ("2016-11-25T00:00:00Z") so `toYMD` strips down to the
    // calendar date the schema actually accepts — without this the
    // submit immediately fails validation on the user's first edit.
    release_date: toYMD(galgame.release_date),
    release_date_tba: galgame.release_date_tba ?? false,
    // U2: deep-clone each row — covers/screenshots are presence-replace,
    // and a shallow ...spread would let the editor mutate the original
    // detail object on add/remove/pin operations (subtle, ugly bugs).
    covers: (galgame.covers ?? []).map((c) => ({ ...c })),
    screenshots: (galgame.screenshots ?? []).map((s) => ({ ...s })),
    // Baseline snapshot so Footer can detect whether the user actually edited
    // the images. Untouched → omit covers/screenshots from the payload so an
    // intro/tag-only edit can't roll the live cover back to this (possibly
    // stale) hydration. Stringified from the same row shape Footer compares.
    covers_baseline: JSON.stringify(
      (galgame.covers ?? []).map((c) => ({ ...c }))
    ),
    screenshots_baseline: JSON.stringify(
      (galgame.screenshots ?? []).map((s) => ({ ...s }))
    ),
    // Creator or admin/moderator → wiki allows direct PUT (instant).
    // Everyone else → PR. Decided here (we have galgame.user + the user
    // store); Footer.vue branches the submit endpoint on this.
    can_direct_edit: galgame.user.id === currentUserId || canModerate.value
  }

  isOpening.value = false
  // ?type=pr arms the `prevent` middleware: on a hard refresh / direct
  // hit the temp store is empty (persist:false) → it redirects out
  // instead of letting the form crash on galgamePR[0]!.
  await navigateTo('/edit/galgame/rewrite?type=pr')
}
</script>

<template>
  <KunTooltip text="编辑">
    <KunButton
      :is-icon-only="true"
      variant="light"
      color="default"
      size="lg"
      :loading="isOpening"
      :disabled="isOpening"
      @click="handleRewriteGalgame(galgame)"
    >
      <KunIcon name="lucide:pencil" />
    </KunButton>
  </KunTooltip>
</template>
