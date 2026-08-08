<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP
} from '~/constants/galgameOfficial'

// `id` is a CATALOG LABEL id (A2-3 / doc 106 R1) — a 会社 is a label in the
// registry's vocabulary — carried bare, with no discriminator segment.
//
// The retired `/galgame-official/` space did need one (`/galgame-official/c/{n}`,
// doc 146): there a bare number was a WIKI id, and wiki/catalog id ranges overlap,
// so a single path serving both meanings would silently render the wrong maker.
// `/galgame/official/` never served wiki ids, so the ambiguity cannot arise. The
// old space survives only as 301 shells (server/middleware/legacy-taxonomy.ts) —
// and unlike tag/engine, its wiki→catalog mapping resolves at RUNTIME through the
// registry's external-ref index rather than from a frozen map.
//
// Staff affordances (edit / delete / revision history) are deliberately NOT
// here any more: those write ops address WIKI rows and this page no longer
// knows a wiki id. They live in the admin taxonomy console, which stays
// wiki-ids end to end (doc 106 R11).
//
// This half of the maker's space is the 会社 ITSELF: identity, the corporate
// family, and a glimpse of the catalogue. Browsing the games — filters, sorts,
// pagination — lives at …/game (see GalgameOfficialDetailNav for why).
const route = useRoute()

// A bookmark or a search result from before the split carries the games grid's
// own state on this URL (`?page=3&sortField=view`), where nothing reads it any
// more — the reader would silently get page 1 of an unfiltered preview. Hand
// them to the page that still answers those keys, before any fetch is paid for.
const carriedFilters = Object.fromEntries(
  Object.entries(route.query).filter(([key]) =>
    (GALGAME_FILTER_QUERY_KEYS as readonly string[]).includes(key)
  )
)
if (Object.keys(carriedFilters).length) {
  await navigateTo(
    { path: `${route.path}/game`, query: route.query },
    { replace: true }
  )
}

// Two rows of the grid at its widest, which is enough to show what a maker
// makes without turning the overview back into the list it just stopped being.
const GALGAME_PREVIEW_LIMIT = 8

const { officialId, data } = await useGalgameOfficialDetail({
  page: 1,
  limit: GALGAME_PREVIEW_LIMIT
})

const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
// SFW mode mirrors the server's IsSFW (cookie showKUNGalgameContentLimit !==
// 'nsfw'): the catalog then hides this maker's r18 works from BOTH the preview
// and the (content-aware) count, so a NSFW-heavy company can look emptier than
// it is. Said in the small print here rather than in a KunInfo banner — the
// full notice lives on the games page, where it is actionable.
const isSfwMode = computed(() => showKUNGalgameContentLimit.value !== 'nsfw')

const gamePath = computed(
  () => `${taxonomyDetailPath('official', officialId)}/game`
)

// An unmapped kind falls through to its raw key: a vocabulary that grew
// upstream should look untranslated, not blank (the map used to be indexed
// bare, so a catalog kind with no Chinese entry rendered an empty chip).
const categoryText = (category: string) =>
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP[category] || category

// A 会社's intro runs to several paragraphs for the makers people actually
// look up, and on a phone that is the whole first screen. So it opens clamped.
//
// The toggle is offered on LENGTH rather than on measured overflow: a
// measurement needs the DOM, which the server render does not have, and a
// button that appears only after hydration moves the text under the reader's
// thumb. Three lines is roughly this many characters at the narrowest layout.
const INTRO_CLAMP_CHARS = 100
const isIntroExpanded = ref(false)

// A tombstone has no name to describe and no URL of its own to be indexed at —
// the survivor's page owns both.
const official = data.value
if (official && !official.moved_to) {
  useKunSeoMeta({
    title: `${official.name} 会社`,
    description: `${official.name}${official.alias?.length ? `, 即 ${official.alias.join('| ')}` : ''}, 查看会社 ${official.name} 制作的所有 Galgame`
  })
}
</script>

<template>
  <div v-if="data && !data.moved_to" class="space-y-6">
    <!-- The intro is NOT the header's description. KunHeader renders that
         immediately under the title at full length, which is exactly the block
         that pushed everything else down; it lives in the body below,
         clamped. -->
    <KunHeader :name="data.name">
      <template v-if="data.logo" #headerEndContent>
        <GalgameOfficialBrandMark :src="data.logo" :name="data.name" />
      </template>

      <template #endContent>
        <div class="space-y-3">
          <!-- One dense line of the 会社's own facts. These used to be three
               stacked rows, each with its own Chinese label in front of a chip;
               the labels said nothing the chip's colour and content did not,
               and they cost three lines of a phone's first screen. -->
          <div class="flex flex-wrap items-center gap-2">
            <KunChip color="primary">{{ categoryText(data.category) }}</KunChip>
            <KunChip v-if="data.lang" color="secondary">
              {{ KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[data.lang] || data.lang }}
            </KunChip>

            <!-- The web presences, on the same line as the facts. Each carries
                 its own name from the server, so an X account is never dressed
                 up as 官方网站, a wikipedia entry says 维基百科 rather than
                 `web` — and a 会社 reachable only on X still gets a way to be
                 reached. -->
            <KunLink
              v-for="link in data.links"
              :key="link.url"
              :is-show-anchor-icon="true"
              target="_blank"
              rel="noopener noreferrer"
              underline="hover"
              size="sm"
              :to="link.url"
            >
              {{ link.name }}
            </KunLink>
          </div>

          <div
            v-if="data.alias.length"
            class="text-default-500 flex flex-wrap items-center gap-2 text-sm"
          >
            别名
            <KunChip size="xs" v-for="(a, index) in data.alias" :key="index">
              {{ a }}
            </KunChip>
          </div>

          <!-- The intro, clamped so the games stay reachable. -->
          <div v-if="data.description" class="space-y-1">
            <p
              :class="
                cn(
                  'text-default-600 whitespace-pre-line',
                  !isIntroExpanded && 'line-clamp-3'
                )
              "
            >
              {{ data.description }}
            </p>
            <KunButton
              v-if="data.description.length > INTRO_CLAMP_CHARS"
              variant="light"
              size="sm"
              color="primary"
              @click="isIntroExpanded = !isIntroExpanded"
            >
              {{ isIntroExpanded ? '收起简介' : '展开简介' }}
            </KunButton>
          </div>
        </div>
      </template>
    </KunHeader>

    <GalgameOfficialDetailNav
      :official-id="officialId"
      :galgame-count="data.galgame_count"
    />

    <!-- The games come FIRST, above the corporate family: they are what a
         reader opens a maker's page for, and the family tree is the block that
         used to bury them. -->
    <div class="space-y-3">
      <KunHeader
        name="作品"
        :description="
          data.galgame_count
            ? `本站已收录 ${data.galgame_count} 部, 下面是最近更新的几部。`
            : ''
        "
        scale="h3"
      />

      <GalgameCard
        v-if="data.galgame.length"
        :is-transparent="false"
        :galgames="data.galgame"
      />

      <KunButton
        v-if="data.galgame_count > GALGAME_PREVIEW_LIMIT"
        variant="flat"
        color="primary"
        :full-width="true"
        :href="gamePath"
      >
        <KunIcon name="lucide:layout-grid" />
        浏览全部 {{ data.galgame_count }} 部作品
      </KunButton>

      <KunNull
        v-if="!data.galgame_count"
        :description="`${data.name} 会社下暂无 Galgame`"
      />
    </div>

    <!-- The corporate family, below the games now. It brings its OWN fetch (see
         the component) and renders nothing at all for a 会社 with no recorded
         relations, which is most of them. -->
    <GalgameOfficialRelationGraph :official-id="officialId" />

    <!-- The page's own small print: provenance, the correction route, and the
         one thing that makes the count above untrustworthy. -->
    <p class="text-default-400 text-xs">
      本页仅展示本站已收录的作品, 资料来自 NextMoe 目录。<template
        v-if="isSfwMode"
        >当前为 SFW 模式, 该会社含 NSFW 内容的 Galgame
        不计入上方数量。</template
      >如果有数据错误请
      <KunLink to="/doc/contact" size="sm">联系我们</KunLink>。
    </p>
  </div>
</template>
