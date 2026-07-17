<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

const props = defineProps<{
  galgame: GalgameDetail
}>()

provide<GalgameDetail>('galgame', props.galgame)

// A ?comment=<id> deep-link (from an @-mention notification) lands on the
// comment tab; the comment section then opens the thread + scrolls to it.
const route = useRoute()
const activeTab = ref(route.query.comment ? 'comment' : 'intro')
const hasPatchResource = ref(false)

// Per-tab loading, surfaced by each async child, so KunTabPanel :loading can dim
// the panel while its data (re)loads (a 2.10.0 feature). intro is static.
const resourceLoading = ref(false)
const patchLoading = ref(false)
const commentLoading = ref(false)
const quizLoading = ref(false)

const contentTabs = computed<KunTabItem[]>(() => [
  { value: 'intro', textValue: '游戏介绍', icon: 'lucide:book-open' },
  { value: 'resource', textValue: '本体资源下载', icon: 'lucide:download' },
  ...(hasPatchResource.value
    ? [{ value: 'patch', textValue: '补丁资源下载', icon: 'lucide:puzzle' }]
    : []),
  { value: 'comment', textValue: '评论区', icon: 'lucide:messages-square' },
  { value: 'quiz', textValue: '题库', icon: 'lucide:brain' }
])

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
    <GalgameHeader
      :galgame="galgame"
      @on-rating-created="handleRatingCreated"
    />

    <!-- Wiki-catalogue game the forum hasn't ingested yet (无本地 galgame 行).
         Explains that any interaction records it, the creator/萌萌点 nuances,
         and only offers 认领 on a claimable VNDB draft (status=2) — a published
         entry (status=0) already has a creator and can't be claimed. -->
    <KunInfo
      v-if="galgame.is_on_forum === false"
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
    <div v-if="galgame.tag?.length" class="md:hidden">
      <GalgameTag :tags="galgame.tag" variant="mobile" />
    </div>

    <div
      v-if="sortedRatings.length && sortedRatings.length >= 3"
      class="grid grid-cols-1 gap-3"
    >
      <GalgameRatingRadarCard :ratings="sortedRatings" />
    </div>

    <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <div class="order-1 flex flex-col gap-3 md:order-2 md:col-span-2">
        <KunScrollShadow>
          <KunTab
            v-model="activeTab"
            :items="contentTabs"
            variant="solid"
            size="md"
            inner-class-name="bg-[oklch(var(--content1))]!"
          />
        </KunScrollShadow>

        <KunCard
          :is-hoverable="false"
          :is-transparent="false"
          content-class="relative"
        >
          <KunTabPanels v-model="activeTab">
            <KunTabPanel value="intro" class-name="space-y-12">
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

              <div v-if="galgame.series" class="space-y-3">
                <KunHeader
                  name="Galgame 系列"
                  description="Galgame 全系列所有 Galgame 作品。例如美少女万华镜 1, 2, 3, 4, 5, 雪女, 外传 就是一个 Galgame 系列"
                  scale="h3"
                />
                <GalgameSeriesCard :series="galgame.series" />
              </div>
            </KunTabPanel>

            <KunTabPanel value="resource" :loading="resourceLoading">
              <GalgameResource @update:loading="resourceLoading = $event" />
            </KunTabPanel>

            <KunTabPanel
              v-if="galgame.vndb_id"
              value="patch"
              :loading="patchLoading"
            >
              <GalgamePatchContainer
                :vndb-id="galgame.vndb_id"
                @has-resource="hasPatchResource = $event"
                @update:loading="patchLoading = $event"
              />
            </KunTabPanel>

            <KunTabPanel value="comment" :loading="commentLoading">
              <GalgameCommentCommunityContainer
                @update:loading="commentLoading = $event"
              />
            </KunTabPanel>

            <KunTabPanel value="quiz" :loading="quizLoading">
              <GalgameQuizGalgamePanel @update:loading="quizLoading = $event" />
            </KunTabPanel>
          </KunTabPanels>
        </KunCard>
      </div>

      <div
        class="order-2 flex flex-col gap-3 md:order-1 md:col-span-1 md:sticky md:top-20 md:self-start"
      >
        <div v-if="galgame.tag?.length" class="hidden md:block">
          <GalgameTag :tags="galgame.tag" variant="desktop" />
        </div>

        <GalgameInfo
          :official="galgame.official"
          :engine="galgame.engine"
          :age-limit="galgame.age_limit"
          :original-language="galgame.original_language"
          :release-date="galgame.release_date"
          :release-date-tba="galgame.release_date_tba"
        />

        <KunCard
          v-if="galgame.contributor?.length"
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
          页面数据全部由
          <KunLink size="sm" target="_blank" to="https://wiki.kungal.com">
            鲲Galgame百科
          </KunLink>
          提供
        </div>
      </div>
    </div>

  </div>
</template>
