<script setup lang="ts">
import { updateGalgameSchema } from '~/validations/galgame'

const { galgamePR } = storeToRefs(useTempGalgamePRStore())

// Creator/admin edit directly (instant); others open a PR. Drives the
// button label here and the endpoint branch in the handler.
const can_direct_edit = computed(
  () => galgamePR.value[0]?.can_direct_edit ?? false
)

const isPublishing = ref(false)

const handlePublishGalgamePR = async () => {
  const galgame = galgamePR.value[0]
  if (!galgame) return

  // Wire-format payload uses snake_case keys to match the wiki PR API
  // (POST /galgame/:gid/prs). Aliases are sent as a string array
  // (replace-all semantics), unlike the create endpoint which takes a
  // comma-separated string — see docs/galgame_wiki/02-revisions-and-prs.md.
  const aliasList = Array.isArray(galgame.alias)
    ? galgame.alias.filter((a) => a.trim().length > 0)
    : String(galgame.alias)
        .split(',')
        .map((a) => a.trim())
        .filter((a) => a.length > 0)

  // Relation arrays are sent as id arrays / {name,link} objects and
  // REPLACE the whole set on merge — Rewrite.vue pre-hydrated them from
  // current data so an untouched section round-trips unchanged. Blank
  // link rows are dropped here so the schema's min(1) doesn't reject
  // them.
  const links = galgame.links
    .map((l) => ({ name: l.name.trim(), link: l.link.trim() }))
    .filter((l) => l.name.length > 0 && l.link.length > 0)

  // Presence-replace for links (same as covers/screenshots): only send/sync
  // links when the user actually changed them vs the hydrated baseline.
  // Untouched → omit → wiki keeps the current set. The crucial case: if the
  // `/link/all` hydration failed, `links` and the baseline are BOTH empty →
  // unchanged → omitted → existing links are preserved, instead of an empty
  // replace-all silently wiping every link of the galgame on merge.
  const linksChanged = JSON.stringify(links) !== (galgame.links_baseline ?? '[]')

  // U2 (K-PR3b): covers/screenshots are presence-replace; strip the
  // server-injected `cdn_url` preview field — wiki doesn't accept it on
  // write (silent-ignored at best, schema-noisy at worst).
  const coversWire = galgame.covers.map((c) => ({
    image_hash: c.image_hash,
    sort_order: c.sort_order,
    sexual: c.sexual,
    violence: c.violence,
    source: c.source,
    source_key: c.source_key
  }))
  const screenshotsWire = galgame.screenshots.map((s) => ({
    image_hash: s.image_hash,
    sort_order: s.sort_order,
    caption: s.caption,
    sexual: s.sexual,
    violence: s.violence,
    source: s.source,
    source_key: s.source_key
  }))

  // Presence-replace semantics: only send covers/screenshots when the user
  // actually edited them (vs the hydrated baseline). Sending an unchanged set
  // re-replaces the live image array with the form's hydration, which rolls
  // the cover back if that hydration was at all stale — the reported "edit
  // intro → cover reverts" bug. Untouched → omit → wiki keeps the current set.
  const coversChanged =
    JSON.stringify(galgame.covers) !== (galgame.covers_baseline ?? '')
  const screenshotsChanged =
    JSON.stringify(galgame.screenshots) !== (galgame.screenshots_baseline ?? '')

  const data: Record<
    string,
    | number
    | string
    | boolean
    | string[]
    | number[]
    | { name: string; link: string }[]
    | Record<string, unknown>[]
  > = {
    vndb_id: galgame.vndb_id,
    name_en_us: galgame.name['en-us'],
    name_ja_jp: galgame.name['ja-jp'],
    name_zh_cn: galgame.name['zh-cn'],
    name_zh_tw: galgame.name['zh-tw'],
    intro_en_us: galgame.introduction['en-us'],
    intro_ja_jp: galgame.introduction['ja-jp'],
    intro_zh_cn: galgame.introduction['zh-cn'],
    intro_zh_tw: galgame.introduction['zh-tw'],
    content_limit: galgame.content_limit,
    age_limit: galgame.age_limit,
    original_language: galgame.original_language,
    // U1: empty = clear to unknown; TBA is independent.
    release_date: galgame.release_date,
    release_date_tba: galgame.release_date_tba,
    aliases: aliasList,
    tag_ids: galgame.tags.map((t) => t.id),
    official_ids: galgame.officials.map((o) => o.id),
    engine_ids: galgame.engines.map((e) => e.id),
    note: galgame.note
  }

  // Only when the user actually touched it (see above) — otherwise the key is
  // absent and the wiki keeps the live links / covers / screenshots.
  if (linksChanged) {
    data.links = links
  }
  if (coversChanged) {
    data.covers = coversWire
  }
  if (screenshotsChanged) {
    data.screenshots = screenshotsWire
  }

  const result = updateGalgameSchema.safeParse(data)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(
      formatKunZodIssue(message),
      'warn'
    )
    return
  }
  // Creator / admin edit directly (PUT /galgame/:gid — instant, new
  // revision). Everyone else opens a PR (POST /:gid/prs). Decided in
  // Rewrite.vue at hydration; see GalgameEditStoreTemp.can_direct_edit.
  const direct = can_direct_edit.value

  const res = await useComponentMessageStore().alert(
    direct ? '确定直接保存对该 Galgame 的修改吗?' : '确定发布 Galgame 信息更新请求吗?',
    direct
      ? '保存后立即生效并写入一条新的版本历史。别名 / 标签 / 会社 / 引擎 / 相关链接 / 封面 / 画廊为整组替换, 请确认上方各项即为最终完整列表。'
      : '别名 / 标签 / 会社 / 引擎 / 相关链接 / 封面 / 画廊为整组替换, 请确认上方各项即为最终完整列表。'
  )
  if (!res) {
    return
  }

  if (isPublishing.value) {
    return
  } else {
    isPublishing.value = true
  }

  // Direct edit → PUT /galgame/:gid; PR → POST /galgame/:gid/prs. JSON
  // only. The legacy "banner as multipart `file`" path was retired in
  // K-PR3b: cover images now upload via /image/galgame and live in the
  // covers[] array (presence-replace). NEW cover/screenshot uploads
  // happen INSIDE the editor (covers/screenshots components), not at
  // submit time, so this Footer has no file-body branch any more.
  const path = direct
    ? `/galgame/${galgame.id}`
    : `/galgame/${galgame.id}/prs`
  const method = direct ? 'PUT' : 'POST'

  const response = await kunFetch(path, { method, body: data })
  isPublishing.value = false

  if (response) {
    // PUT /galgame/:gid does NOT process aliases/links (01-galgame.md);
    // they live on dedicated add/remove endpoints. The PR endpoint DOES
    // take them (replace-all), so only the direct path needs to
    // reconcile aliases/links itself against the current set.
    if (direct) {
      await reconcileAliasesLinks(
        galgame.id,
        galgame.alias,
        galgame.links,
        linksChanged
      )
    }
    useKunLoliInfo(direct ? '已保存, 修改已生效' : '创建更新请求成功', 5)
    await navigateTo(`/galgame/${galgame.id}`, {
      replace: true
    })
  }
}

// Reconcile aliases & links to the edited set via the dedicated
// per-item endpoints (docs 03-relations): GET current → POST the added,
// DELETE the removed (each creates its own revision). Best-effort: a
// failure warns but doesn't undo the already-applied PUT.
const reconcileAliasesLinks = async (
  gid: number,
  desiredAliasRaw: string[],
  desiredLinksRaw: { name: string; link: string }[],
  // Skip the link diff when the user didn't change links (esp. when hydration
  // failed → desiredLinksRaw is empty). Without this gate the diff would DELETE
  // every current link, the direct-edit twin of the PR empty-replace wipe.
  syncLinks: boolean
) => {
  let failed = false

  // aliases
  const curAliases =
    (await kunFetch<{ id: number; name: string }[]>(
      `/galgame/${gid}/aliases`,
      { method: 'GET' }
    )) ?? []
  const desiredAlias = [
    ...new Set(desiredAliasRaw.map((a) => a.trim()).filter(Boolean))
  ]
  for (const name of desiredAlias) {
    if (!curAliases.some((a) => a.name === name)) {
      const r = await kunFetch(`/galgame/${gid}/aliases`, {
        method: 'POST',
        body: { name }
      })
      if (r === null) failed = true
    }
  }
  for (const a of curAliases) {
    if (!desiredAlias.includes(a.name)) {
      const r = await kunFetch(`/galgame/${gid}/aliases`, {
        method: 'DELETE',
        body: { id: a.id }
      })
      if (r === null) failed = true
    }
  }

  // links — identity is the (name, link) pair. Gated: an untouched / failed-to-
  // hydrate set must NOT reconcile (it would delete every current link).
  if (syncLinks) {
    const curLinks =
      (await kunFetch<{ id: number; name: string; link: string }[]>(
        `/galgame/${gid}/links`,
        { method: 'GET' }
      )) ?? []
    const key = (n: string, l: string) => JSON.stringify([n, l])
    const desiredLinks = desiredLinksRaw
      .map((l) => ({ name: l.name.trim(), link: l.link.trim() }))
      .filter((l) => l.name && l.link)
    const desiredSet = new Set(desiredLinks.map((l) => key(l.name, l.link)))
    const curSet = new Set(curLinks.map((l) => key(l.name, l.link)))
    for (const l of desiredLinks) {
      if (!curSet.has(key(l.name, l.link))) {
        const r = await kunFetch(`/galgame/${gid}/links`, {
          method: 'POST',
          body: { name: l.name, link: l.link }
        })
        if (r === null) failed = true
      }
    }
    for (const l of curLinks) {
      if (!desiredSet.has(key(l.name, l.link))) {
        const r = await kunFetch(`/galgame/${gid}/links`, {
          method: 'DELETE',
          body: { id: l.id }
        })
        if (r === null) failed = true
      }
    }
  }

  if (failed) {
    useMessage('部分别名 / 链接未能保存, 请到该条目核对后重试', 'warn', 5000)
  }
}
</script>

<template>
  <div class="flex justify-end">
    <KunButton
      :disabled="isPublishing"
      :loading="isPublishing"
      size="lg"
      @click="handlePublishGalgamePR"
    >
      {{ can_direct_edit ? '保存修改' : '提交更新请求' }}
    </KunButton>
  </div>
</template>
