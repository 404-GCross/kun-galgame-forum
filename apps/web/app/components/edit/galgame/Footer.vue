<script setup lang="ts">
import { submitGalgameSchema } from '~/validations/galgame'

// After the submission flow landed (docs/galgame_wiki/07-submission.md),
// regular users go through POST /galgame/submit which creates a status=3
// draft awaiting moderation. The legacy POST /galgame endpoint is now
// gated to admin/moderator at both the wiki and kungal layers, so calling
// it as a normal user returns 403. The post-success redirect goes to
// /edit/galgame/mine — status=3 isn't anonymously visible, so /galgame/:gid
// would 404 for the submitter until approval.

// No vndb_id: submission is exclusively for VNDB-unlisted works (wiki has
// the full VNDB set as claimable drafts already) — see Galgame.vue.
const {
  name,
  content_limit,
  age_limit,
  original_language,
  introduction,
  aliases,
  release_date,
  release_date_tba
} = storeToRefs(usePersistEditGalgameStore())

const isPublishing = ref(false)

const handleSubmitGalgame = async () => {
  const banner = await getImage('kun-galgame-publish-banner')
  // Wire-format payload uses snake_case keys to match the wiki API
  // (POST /galgame/submit). The Vue store keeps camelCase locally; we
  // rename at the boundary so the schema, the JSON body, and the wiki
  // contract all agree.
  const data: Record<
    string,
    number | string | string[] | Blob | boolean | null
  > = {
    name_en_us: name.value['en-us'],
    name_ja_jp: name.value['ja-jp'],
    name_zh_cn: name.value['zh-cn'],
    name_zh_tw: name.value['zh-tw'],
    intro_en_us: introduction.value['en-us'],
    intro_ja_jp: introduction.value['ja-jp'],
    intro_zh_cn: introduction.value['zh-cn'],
    intro_zh_tw: introduction.value['zh-tw'],
    content_limit: content_limit.value,
    age_limit: age_limit.value,
    original_language: original_language.value,
    // U1: empty string = unknown; wiki schema accepts "" or YYYY-MM-DD.
    release_date: release_date.value,
    release_date_tba: release_date_tba.value,
    banner,
    aliases: String(aliases.value)
  }
  const result = submitGalgameSchema.safeParse(data)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(
      formatKunZodIssue(message),
      'warn'
    )
    return
  }
  const res = await useComponentMessageStore().alert(
    '确定提交 Galgame 申请吗?',
    '提交后将进入审核队列, 审核通过后才会被公开展示。审核结果会通过站内消息通知您。在审核期间您可以在「我的提交」页继续编辑或撤回。'
  )
  if (!res) {
    return
  }

  if (isPublishing.value) {
    return
  } else {
    isPublishing.value = true
    useMessage(10525, 'info', 7777)
  }

  // multipart 上传约定参见 docs/galgame_wiki/01-galgame.md "Banner 上传":
  //   data: 整个 JSON 串
  //   file: 可选图片二进制
  const { banner: _bannerBlob, ...jsonFields } = data
  const formData = new FormData()
  formData.append('data', JSON.stringify(jsonFields))
  if (banner instanceof Blob) {
    formData.append('file', banner)
  }
  // POST /galgame/submit returns the created draft (`{id, status: 3, ...}`).
  // We only need `id` to confirm success — the redirect goes to the
  // submitter's pending list, not the public detail page.
  const created = await kunFetch<{ id: number; status: number }>(
    '/galgame/submit',
    {
      method: 'POST',
      body: formData
    }
  )
  isPublishing.value = false

  if (created?.id) {
    await deleteImage('kun-galgame-publish-banner')
    // Clear the persisted wizard-step draft key (set by Galgame.vue via
    // useLocalStorage) so the next "new submission" starts at step ①
    // rather than resuming a stale position over empty fields.
    if (import.meta.client) {
      localStorage.removeItem('kun-galgame-publish-step')
    }

    useKunLoliInfo('Galgame 申请已提交, 等待审核', 5)
    await navigateTo('/edit/galgame/mine')
    usePersistEditGalgameStore().resetEditGalgameStore()
  }
}
</script>

<template>
  <div class="flex justify-end">
    <KunButton
      :disabled="isPublishing"
      :loading="isPublishing"
      size="lg"
      @click="handleSubmitGalgame"
    >
      提交 Galgame 申请
    </KunButton>
  </div>
</template>
