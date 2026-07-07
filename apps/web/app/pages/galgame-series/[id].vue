<script setup lang="ts">
// Series detail shows revision history (the "编辑历史" button in
// GalgameSeriesDetail) — its name/alias/description edits. Membership
// changes (a galgame joining/leaving) are recorded as galgame-side
// revisions, so they show on each galgame's history rather than here.
const route = useRoute()

const seriesId = computed(() => {
  return Number((route.params as { id: string }).id)
})

const { data } = await useKunFetch<GalgameSeriesDetail>(`/galgame-series/${seriesId.value}`, {
  method: 'GET',
  query: { series_id: seriesId.value }
})

if (data.value) {
  if (data.value.is_nsfw) {
    useKunDisableSeo(data.value.name)
  } else {
    const seriesBanner =
      data.value.sample_galgame[0]?.effective_banner_url ??
      data.value.sample_galgame[0]?.banner
    useKunSeoMeta({
      title: `${data.value.name} 系列下载资源`,
      description:
        data.value.description ||
        `${data.value.name} 系列收录的全部 Galgame 作品、剧情脉络与下载资源。`,
      ...(seriesBanner ? { ogImage: seriesBanner } : {})
    })
  }
} else {
  useKunDisableSeo('未找到 Galgame 系列')
}
</script>

<template>
  <div>
    <!-- Single real root box, NOT `display: contents` (see user.vue): a box-less
         root trips Nuxt's "does not have a single root node" and skips the
         page-transition enter, so the page teleports in. -->
    <GalgameSeriesDetail :data="data" v-if="data" />
    <KunNull v-else description="未找到这个 Galgame 系列" />
  </div>
</template>
