<script setup lang="ts">
import type {
  VideoGame,
  WithContext,
  Person,
  CreativeWorkSeries
} from 'schema-dts'

const userId = storeToRefs(usePersistUserStore()).id.value
const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
// Key by path so navigating between two items of this dynamic route remounts
// the page and re-runs setup — the detail fetch uses a static URL + watch:false.
definePageMeta({ key: (route) => route.path })

const route = useRoute()

// "NSFW mode" = cookie says nsfw or all. Used together with `userId` to
// decide whether to short-circuit the NSFW interstitial:
//   logged-in           → show directly (better UX for known visitors)
//   anonymous + NSFW on → show directly (the user opted in)
//   anonymous + SFW     → confirm interstitial (default browser policy)
const isNsfwMode = computed(
  () =>
    showKUNGalgameContentLimit.value === 'nsfw' ||
    showKUNGalgameContentLimit.value === 'all'
)

const gid = computed(() => {
  return parseInt((route.params as { gid: string }).gid)
})

const { data } = await useKunFetch<GalgameDetail>(`/galgame/${gid.value}`, {
  method: 'GET',
  watch: false,
  query: { galgameId: gid.value }
})

const galgame = data.value
const isShowGalgame = ref(true)

if (galgame) {
  if (galgame.content_limit === 'nsfw') {
    const title = getPreferredLanguageText(galgame.name)
    // Disable SEO meta either way — NSFW pages should never feed
    // OpenGraph / rich-result hints to crawlers, regardless of who's
    // looking. Title is suppressed entirely for the SFW-cookie anonymous
    // case so even the document.title can't leak.
    const trustedVisitor = !!userId || isNsfwMode.value
    useKunDisableSeo(trustedVisitor ? title : '')

    if (!trustedVisitor) {
      isShowGalgame.value = false
    }
  } else {
    const titleBase = getPreferredLanguageText(galgame.name)
    const jaTitle = galgame.name['ja-jp']
    const title =
      jaTitle && titleBase !== jaTitle ? `${titleBase} | ${jaTitle}` : titleBase
    const pageUrl = `${kungal.domain.main}${route.path}`
    const description = markdownToText(
      getPreferredLanguageText(galgame.markdown)
    )
      .slice(0, 175)
      .replace(/\\|\n/g, '')

    const jsonLd: WithContext<VideoGame> = {
      '@context': 'https://schema.org',
      '@type': 'VideoGame',
      name: titleBase,
      alternateName: galgame.alias,
      url: pageUrl,
      image: getEffectiveBanner(galgame),
      description: description,
      inLanguage: galgame.original_language,
      datePublished: new Date(galgame.created).toISOString(),
      dateModified: new Date(galgame.updated).toISOString(),
      publisher: galgame.official.map((o) => ({
        '@type': 'Organization',
        name: o.name
      })),

      genre: galgame.tag
        .filter((t) => t.category === 'content')
        .map((t) => t.name),
      keywords: galgame.tag
        .filter((t) => t.category === 'technical')
        .map((t) => t.name)
        .join(', '),

      ...(galgame.series && {
        isPartOf: {
          '@type': 'CreativeWorkSeries',
          name: galgame.series.name,
          url: `${kungal.domain.main}/series/${galgame.series.id}`
        } satisfies CreativeWorkSeries
      }),

      interactionStatistic: [
        {
          '@type': 'InteractionCounter',
          interactionType: {
            '@type': 'LikeAction'
          },
          userInteractionCount: galgame.like_count
        },
        {
          '@type': 'InteractionCounter',
          interactionType: {
            '@type': 'WatchAction'
          },
          userInteractionCount: galgame.view
        }
      ],

      author: {
        '@type': 'Person',
        name: galgame.user.name
      } satisfies Person,
      contributor: galgame.contributor.map((c) => ({
        '@type': 'Person',
        name: c.name
      })) satisfies Person[]
    }

    useHead({
      script: [
        {
          id: 'schema-org-video-game',
          type: 'application/ld+json',
          innerHTML: jsonLd
        }
      ]
    })

    useKunSeoMeta({
      title,
      description,
      ogImage: getEffectiveBanner(galgame),
      articleAuthor: [`${kungal.domain.main}/user/${galgame.user.id}`],
      articlePublishedTime: galgame.created.toString(),
      articleModifiedTime: galgame.updated.toString()
    })
  }
} else {
  useKunDisableSeo('请求 Galgame 错误')
}
</script>

<template>
  <div>
    <div v-if="data">
      <Galgame v-if="isShowGalgame" :galgame="data" />

      <KunCard v-else :is-hoverable="false" :is-transparent="false">
        <p>这个 Galgame 含有 NSFW 内容, 您需要点击确认以显示这个 Galgame</p>
        <KunButton @click="isShowGalgame = true">确认显示</KunButton>
      </KunCard>
    </div>

    <KunNull v-else description="未找到这个 Galgame" />
  </div>
</template>
