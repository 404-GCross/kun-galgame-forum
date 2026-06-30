<script setup lang="ts">
const props = defineProps<{
  galgame: GalgameDetail
}>()

provide<GalgameDetail>('galgame', props.galgame)

// One shared "编辑历史 / 更新请求" modal for the whole detail page. Info.vue's
// button opens it on 编辑历史; the pending-PR banner (GalgamePrBanner, creator/
// admin only) deep-links to 更新请求. Owned here — the one place that has
// `galgame` — so both triggers drive a SINGLE modal instance (no duplicate
// PR/history fetches, no async-mount-without-Suspense risk).
const activity = reactive<{ open: boolean; tab: 'history' | 'pr' }>({
  open: false,
  tab: 'history'
})
provide('galgameActivity', activity)

const ratings = ref([...props.galgame.ratings])
const sortedRatings = computed(() => {
  return [...ratings.value].sort(
    (a, b) => b.short_summary.length - a.short_summary.length
  )
})

const handleRatingCreated = (newRating: GalgameRatingCardOnGalgamePage) => {
  ratings.value.unshift(newRating)
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <!-- Creator/admin-only; renders nothing (no flex gap) for everyone else. -->
    <GalgamePrBanner :galgame="galgame" />

    <GalgameHeader
      :galgame="galgame"
      @on-rating-created="handleRatingCreated"
    />

    <!-- Wiki-catalogue game the forum hasn't ingested yet (无本地 galgame 行).
         Explains that any interaction records it, the creator/萌萌点 nuances,
         and only offers 认领 on a claimable VNDB draft (status=2) — a published
         entry (status=0) already has a creator and can't be claimed. -->
    <KunInfo
      v-if="galgame.isOnForum === false"
      color="danger"
      title="该游戏尚未在本站收录"
      description="本站还没有这款 Galgame 的任何本地数据, 当前页面的资料均来自百科。点赞 / 收藏 / 评论 / 评分 都会让它被本站收录, 但您不会成为该 Galgame 的创建者, 也不会获得萌萌点奖励; 发布下载资源同样会让它被收录, 并照常获得发布资源的萌萌点奖励。"
    >
      <p v-if="galgame.status === 2" class="text-sm">
        想成为该 Galgame 的创建者? 可前往「发布
        Galgame」页面认领它（认领后您将成为创建者并获得萌萌点奖励）。
      </p>
    </KunInfo>

    <!-- Mobile: tags sit right under the header. On desktop they live at the top
         of the sidebar instead (the stacked single-column layout would otherwise
         push them below all the main content). Two breakpoint-gated instances —
         see GalgameTag's `variant`. -->
    <div class="md:hidden">
      <GalgameTag :tags="galgame.tag" variant="mobile" />
    </div>

    <div
      v-if="sortedRatings.length && sortedRatings.length >= 3"
      class="grid grid-cols-1 gap-3"
    >
      <GalgameRatingRadarCard :ratings="sortedRatings" />
    </div>

    <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <div class="md:col-span-2">
        <KunCard
          :is-hoverable="false"
          :is-transparent="false"
          content-class="space-y-12 relative"
        >
          <div class="space-y-3">
            <GalgameIntroduction :introduction="galgame.introduction" />

            <div
              v-if="sortedRatings.length && sortedRatings.length < 3"
              class="space-y-1"
            >
              <GalgameRatingRow
                v-for="rating in sortedRatings"
                :key="rating.id"
                :rating="rating"
              />
            </div>

            <GalgameLink />
          </div>

          <GalgameGallery :screenshots="galgame.screenshots" />

          <GalgameResource />

          <GalgamePatchContainer :vndb-id="galgame.vndbId" />

          <div v-if="galgame.series" class="space-y-3">
            <KunHeader
              name="Galgame 系列"
              description="Galgame 全系列所有 Galgame 作品。例如美少女万华镜 1, 2, 3, 4, 5, 雪女, 外传 就是一个 Galgame 系列"
              scale="h3"
            />
            <GalgameSeriesCard :series="galgame.series" />
          </div>

          <GalgameCommentContainer
            :user-data="galgame.contributor"
            :target-user="galgame.user"
          />
        </KunCard>
      </div>

      <div class="flex flex-col gap-3 md:col-span-1">
        <div class="hidden md:block">
          <GalgameTag :tags="galgame.tag" variant="desktop" />
        </div>

        <GalgameInfo
          :official="galgame.official"
          :engine="galgame.engine"
          :age-limit="galgame.ageLimit"
          :original-language="galgame.originalLanguage"
          :release-date="galgame.releaseDate"
          :release-date-tba="galgame.releaseDateTBA"
        />

        <KunCard
          content-class="space-y-3"
          :is-hoverable="false"
          :is-transparent="false"
        >
          <KunHeader
            name="贡献者"
            description="本游戏项目的贡献者, 计 Galgame 资源发布贡献"
            scale="h3"
          />

          <div
            class="text-default-500 flex cursor-default flex-wrap items-center gap-2"
          >
            <KunUserChip :user="galgame.user" />
            <span class="text-sm">
              <KunTime :time="galgame.created" type="date" show-year />
              创建本游戏
            </span>
          </div>

          <GalgameContributorContainer />
        </KunCard>

        <div class="text-default-500 flex items-center justify-center text-sm">
          部分页面数据由
          <KunLink size="sm" target="_blank" to="https://vndb.org">
            VNDB
          </KunLink>
          提供
        </div>
      </div>
    </div>

    <GalgameActivityModal v-model="activity.open" :initial-tab="activity.tab" />
  </div>
</template>
