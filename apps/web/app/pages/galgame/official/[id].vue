<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP,
  KUN_GALGAME_OFFICIAL_LINK_SOURCE_MAP
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
const route = useRoute()
const official_id = computed(() => {
  return Number((route.params as { id: string }).id)
})

// A junk segment (/galgame/official/null, crawler-made) becomes NaN and used to
// ride all the way upstream, where the catalog answered 400 — pointless round
// trips for a URL that can only ever be a 404. Answer it here.
if (!Number.isInteger(official_id.value) || official_id.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 会社',
    fatal: true
  })
}

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

const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())
// SFW mode mirrors the server's IsSFW (cookie showKUNGalgameContentLimit !==
// 'nsfw'): the catalog then hides this maker's r18 works from BOTH the list and
// the (content-aware) count, so a NSFW-heavy company can look emptier than it is.
const isSfwMode = computed(() => showKUNGalgameContentLimit.value !== 'nsfw')

// "未发布的游戏": catalog works by this maker that no product has an entry for
// yet. Public claim funnel — open to everyone, not just moderators.
const showDraftsModal = ref(false)

// An unmapped kind falls through to its raw key: a vocabulary that grew
// upstream should look untranslated, not blank (the map used to be indexed
// bare, so a catalog kind with no Chinese entry rendered an empty chip).
const categoryText = (category: string) =>
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP[category] || category
const linkText = (source: string) =>
  KUN_GALGAME_OFFICIAL_LINK_SOURCE_MAP[source] || source

// A 会社's intro runs to several paragraphs for the makers people actually
// look up, and on a phone that is the whole first screen — the games, which is
// what the page is FOR, start below the fold. So it opens clamped.
//
// The toggle is offered on LENGTH rather than on measured overflow: a
// measurement needs the DOM, which the server render does not have, and a
// button that appears only after hydration moves the text under the reader's
// thumb. Three lines is roughly this many characters at the narrowest layout.
const INTRO_CLAMP_CHARS = 100
const isIntroExpanded = ref(false)

const { data, status } = await useKunFetch<GalgameOfficialDetail>(
  `/galgame-official/${official_id.value}`,
  {
    method: 'GET',
    query: {
      page,
      limit,
      type,
      language,
      platform,
      gameType,
      sortField,
      sortOrder,
      official_id
    }
  }
)

// A merged 会社 keeps its old id addressable, but only as a 301: the catalog
// merges duplicate labels, and the loser's id has to land on the survivor's
// page rather than render a copy of it — two URLs for one company is exactly
// what the redirect prevents. `moved_to` arrives instead of the record, never
// alongside it, so nothing of the survivor is ever painted under the old id.
// The target is built from the shared path builder, so it is the FINAL form
// and the hop can never become a chain.
//
// `navigateTo` is NOT an early return here, awaited or not: on the server it
// only parks a redirect on ssrContext['~renderResponse'] and hands control
// back, so the rest of setup AND the template still render — against the
// tombstone payload (id 0, alias/galgame null). A throw from that render
// preempts the parked redirect and the visitor gets a 500 instead of the 301
// (prod: /galgame/official/13323 died on `alias.length`). So everything below
// that touches the record is gated on `moved`, and the template root carries
// the same gate as `!data.moved_to` — reactive, so a client-side hop stops
// painting the dead brand too.
const moved = !!data.value?.moved_to
if (data.value?.moved_to) {
  await navigateTo(taxonomyDetailPath('official', data.value.moved_to), {
    redirectCode: 301,
    replace: true
  })
}

// An unknown id is a real 404, not an empty 200 shell: this namespace went live
// with no legacy id space behind it, so a miss means the entity does not exist
// and a crawler must be told exactly that rather than indexing a blank page.
if (!data.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 会社',
    fatal: true
  })
}

// A tombstone has no name to describe and no URL of its own to be indexed at —
// the survivor's page owns both.
if (!moved) {
  useKunSeoMeta({
    title: `${data.value.name} 会社`,
    description: `${data.value.name}${data.value.alias?.length ? `, 即 ${data.value.alias.join('| ')}` : ''}, 查看会社 ${data.value.name} 制作的所有 Galgame`
  })
}
</script>

<template>
  <div v-if="data && !data.moved_to" class="space-y-6">
    <!-- The intro is NOT the header's description any more. KunHeader renders
         that immediately under the title at full length, which is exactly the
         block that pushed the games off a phone screen; it now lives in the
         body below, clamped. -->
    <KunHeader :name="`${data.name} 制作的 Galgame`">
      <!-- The 会社's own brand mark, at full size (the header frame is large
           enough that the 360px `_mini` variant would show its resampling).
           Logos are always SFW upstream, so no content gate sits in front of
           it. A maker with none renders no frame at all — the header simply
           looks the way it always has. -->
      <template v-if="data.logo" #headerEndContent>
        <!-- On its own light surface, not bare: a brand mark is usually a
             transparent PNG drawn in one dark colour, and against the dark
             theme's background half of them disappear entirely. -->
        <div class="bg-default-100 shrink-0 rounded-lg p-2 sm:p-3">
          <KunImage
            :src="data.logo"
            :alt="`${data.name} logo`"
            loading="eager"
            object-fit="contain"
            class-name="size-14 sm:size-20"
          />
        </div>
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
            <KunChip v-if="data.galgame_count" color="success">
              本站收录 {{ data.galgame_count }} 部
            </KunChip>

            <!-- The web presences, on the same line as the facts. Each is named
                 by its own source, so an X account is never dressed up as
                 官方网站 — and a 会社 reachable only on X still gets a way to
                 be reached. -->
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
              {{ linkText(link.source) }}
            </KunLink>

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

          <!-- The page's own small print. It was four sentences of body text at
               the very top; two of them (the SFW rule, the filters) are already
               said by the KunInfo and the filter bar right below it, so what is
               left is the provenance and the correction route. -->
          <p class="text-default-400 text-xs">
            本页仅展示本站已收录的作品, 资料来自 NextMoe 目录。如果有数据错误请
            <KunLink to="/doc/contact" size="sm">联系我们</KunLink>。
          </p>
        </div>
      </template>
    </KunHeader>

    <!-- The corporate family, between the header and the games. It brings its
         OWN fetch (see the component): this page's payload is refetched on
         every pagination / filter change below, and none of that can move a
         company under a different parent. Renders nothing at all for a 会社
         with no recorded relations, which is most of them. -->
    <GalgameOfficialRelationGraph :official-id="official_id" />

    <GalgameCardNav :show-advanced="false" />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该会社含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <GalgameDraftsModal
      v-model="showDraftsModal"
      entity-type="official"
      :entity-id="official_id"
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
