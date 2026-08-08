<script setup lang="ts">
// The 会社's catalogue, on its own URL. Everything on this page serves one job
// — finding a game this maker made — so the filter bar sits directly under the
// title with the grid right below it, instead of being sandwiched between the
// maker's intro and a corporate family tree (see GalgameOfficialDetailNav).
//
// `id` is a CATALOG LABEL id, carried bare; the id-space reasoning and the
// merged-id 301 both live in useGalgameOfficialDetail, shared with the
// overview.
const {
  page,
  limit,
  type,
  language,
  platform,
  gameType,
  sortField,
  sortOrder
} = useGalgameFilters()

const { officialId, data, status } = await useGalgameOfficialDetail(
  {
    page,
    limit,
    type,
    language,
    platform,
    gameType,
    sortField,
    sortOrder
  },
  '/game'
)

const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
// SFW mode mirrors the server's IsSFW (cookie showKUNGalgameContentLimit !==
// 'nsfw'): the catalog then hides this maker's r18 works from BOTH the list and
// the (content-aware) count, so a NSFW-heavy company can look emptier than it
// is.
const isSfwMode = computed(() => showKUNGalgameContentLimit.value !== 'nsfw')

// "未发布的游戏": catalog works by this maker that no product has an entry for
// yet. Public claim funnel — open to everyone, not just moderators. It belongs
// here rather than on the overview: it is another way of browsing the same
// catalogue.
const showDraftsModal = ref(false)

const official = data.value
if (official && !official.moved_to) {
  useKunSeoMeta({
    title: `${official.name} 制作的 Galgame`,
    description: `浏览会社 ${official.name} 制作的全部 Galgame, 可按类型 / 语言 / 平台 / 作品类型筛选与排序。`
  })
}
</script>

<template>
  <div v-if="data && !data.moved_to" class="space-y-6">
    <KunHeader :name="`${data.name} 制作的 Galgame`">
      <template v-if="data.logo" #headerEndContent>
        <GalgameOfficialBrandMark
          :src="data.logo"
          :name="data.name"
          size="md"
        />
      </template>

      <template #endContent>
        <!-- No count chip here: the 作品 tab right below already carries it,
             and printing the same number twice in adjacent rows reads as two
             different facts. -->
        <div class="flex flex-wrap items-center gap-2">
          <KunButton
            class-name="ml-auto"
            variant="flat"
            size="sm"
            color="default"
            @click="showDraftsModal = true"
          >
            <KunIcon name="lucide:library-big" />
            未发布的游戏
          </KunButton>
        </div>
      </template>
    </KunHeader>

    <GalgameOfficialDetailNav
      :official-id="officialId"
      :galgame-count="data.galgame_count"
    />

    <GalgameCardNav :is-show-advanced="false" />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该会社含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <GalgameDraftsModal
      v-model="showDraftsModal"
      entity-type="official"
      :entity-id="officialId"
      :entity-name="data.name"
    />

    <GalgameCard
      :is-transparent="false"
      v-if="data.galgame.length"
      :galgames="data.galgame"
    />

    <KunPagination
      v-if="data.galgame_count > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.galgame_count / limit)"
      :is-loading="status === 'pending'"
    />

    <KunNull
      v-if="!data.galgame_count"
      :description="`${data.name} 会社下暂无 Galgame`"
    />
  </div>
</template>
