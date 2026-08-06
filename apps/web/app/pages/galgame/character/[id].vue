<script setup lang="ts">
// `id` is a CATALOG character id — the id the 登场角色 roster on every game
// detail page already carries, so the link needs no lookup.
//
// The page answers the question the game-page popup deliberately does not: not
// "who is she" but "where else have I seen her". Everything below the header is
// the appearance list, rendered through the site's ordinary galgame card.
//
// What is NOT here is as deliberate: the catalog's public lane publishes no
// 性别 / 生日 / 三围 for a character (those live on the staff-side face only),
// so this page does not have them and will not invent them from elsewhere.
const route = useRoute()
const characterId = computed(() => Number((route.params as { id: string }).id))

if (!Number.isInteger(characterId.value) || characterId.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该角色',
    fatal: true
  })
}

const PAGE_SIZE = 50

const { data } = await useKunFetch<GalgameCharacterDetail>(
  `/galgame-character/${characterId.value}`,
  { method: 'GET', query: { limit: PAGE_SIZE }, watch: false }
)

// A merged character keeps its old id addressable, but only as a 301:
// `moved_to` arrives instead of the record, never alongside it. `navigateTo` is
// NOT an early return on the server — it parks the redirect and hands control
// back — so everything below that touches the record is gated on `moved`, and
// so is the template root (the moved payload's works/traits are null, and an
// ungated spread would 500 over the parked 301).
const moved = !!data.value?.moved_to
if (data.value?.moved_to) {
  await navigateTo(`/galgame/character/${data.value.moved_to}`, {
    redirectCode: 301,
    replace: true
  })
}

if (!data.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该角色',
    fatal: true
  })
}

// The appearance list is offset-paged with no total, so this is a 加载更多
// rather than a pager: we know whether ANOTHER page exists, never how many.
const works = ref<GalgameCharacterWork[]>(moved ? [] : [...data.value.works])
const nextOffset = ref<number | null>(moved ? null : data.value.next_offset)
const loadingMore = ref(false)

const loadMore = async () => {
  if (nextOffset.value === null || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const res = await kunFetch<GalgameCharacterDetail>(
    `/galgame-character/${characterId.value}`,
    { method: 'GET', query: { limit: PAGE_SIZE, offset: nextOffset.value } }
  )
  loadingMore.value = false
  if (!res) {
    return
  }
  works.value.push(...res.works)
  nextOffset.value = res.next_offset
}

const subtitle = computed(() => {
  const parts = [data.value?.name_zh, data.value?.latin].filter(
    (part): part is string => !!part && part !== data.value?.name
  )
  return parts.join(' · ')
})

// The header art: the 立绘 when there is one, because a standing figure is the
// picture of a character — the bust then rides beside it at thumbnail size.
// Both are opened full-size by the lightbox; neither is cropped into the
// other's frame.
const isMachineIntro = computed(
  () => !!data.value?.intros.find((i) => i.intro === data.value?.intro)?.machine
)
const otherIntros = computed(() =>
  (data.value?.intros ?? []).filter((i) => i.intro !== data.value?.intro)
)

// Spoiler traits ride along with the response and wait for one explicit click.
const isTraitSpoilerRevealed = ref(false)
const traits = computed(() => {
  const all = data.value?.traits ?? []
  return isTraitSpoilerRevealed.value ? all : all.filter((t) => t.spoiler === 0)
})
const hiddenTraitCount = computed(
  () => (data.value?.traits ?? []).filter((t) => t.spoiler > 0).length
)
const traitGroups = computed(() => {
  const groups: { name: string; traits: GalgameCharacterTrait[] }[] = []
  for (const trait of traits.value) {
    const name = trait.group || '其他'
    const last = groups.at(-1)
    if (last && last.name === name) {
      last.traits.push(trait)
    } else {
      groups.push({ name, traits: [trait] })
    }
  }
  return groups
})

if (!moved) {
  useKunSeoMeta({
    title: `${data.value.name} 登场的 Galgame`,
    description:
      data.value.intro ||
      `角色 ${data.value.name} 在本站收录的 Galgame 中的登场作品与配音演员一览。`
  })
}
</script>

<template>
  <div v-if="data && !data.moved_to" class="space-y-6">
    <KunHeader :name="data.name" :description="subtitle">
      <template v-if="data.figure || data.image" #headerEndContent>
        <KunLightboxGallery>
          <div class="flex shrink-0 items-start gap-2">
            <!-- Contained on its own surface: a 立绘 is cut out against a flat
                 field and arrives at whatever ratio the source drew it, so the
                 leftover has to be background rather than a crop. -->
            <KunLightboxGalleryItem
              v-if="data.figure"
              :src="data.figure"
              :alt="data.name"
              :wrap="false"
              v-slot="{ open }"
            >
              <button
                type="button"
                class="bg-default-100 cursor-zoom-in overflow-hidden rounded-lg"
                :aria-label="`查看 ${data.name} 的立绘`"
                @click="open"
              >
                <KunImage
                  :src="data.figure"
                  :alt="data.name"
                  loading="eager"
                  aspect-ratio="1/1"
                  object-fit="contain"
                  class-name="w-40 sm:w-48"
                />
              </button>
            </KunLightboxGalleryItem>

            <KunLightboxGalleryItem
              v-if="data.image"
              :src="data.image"
              :alt="data.name"
              :wrap="false"
              v-slot="{ open }"
            >
              <button
                type="button"
                class="bg-default-100 cursor-zoom-in overflow-hidden rounded-lg"
                :aria-label="`查看 ${data.name} 的头像`"
                @click="open"
              >
                <KunImage
                  :src="data.image"
                  :alt="data.name"
                  loading="eager"
                  aspect-ratio="3/4"
                  object-fit="cover"
                  :class-name="data.figure ? 'w-24 sm:w-28' : 'w-32 sm:w-40'"
                />
              </button>
            </KunLightboxGalleryItem>
          </div>
        </KunLightboxGallery>
      </template>

      <template #endContent>
        <div class="space-y-3">
          <div v-if="data.intro" class="space-y-1">
            <p class="text-default-600 whitespace-pre-line">{{ data.intro }}</p>
            <p v-if="isMachineIntro" class="text-default-400 text-xs">
              本段简介由机器翻译生成
            </p>
          </div>

          <!-- The other languages, collapsed. One bio is what a reader wants;
               the originals are what a reader occasionally wants. -->
          <KunAccordion v-if="otherIntros.length">
            <KunAccordionItem
              v-for="intro in otherIntros"
              :key="intro.lang"
              :value="intro.lang"
              :title="`${intro.lang} 简介`"
            >
              <p class="text-default-600 whitespace-pre-line">
                {{ intro.intro }}
              </p>
            </KunAccordionItem>
          </KunAccordion>

          <div v-if="traitGroups.length" class="space-y-2">
            <div
              v-for="group in traitGroups"
              :key="group.name"
              class="space-y-1"
            >
              <p class="text-default-400 text-xs">{{ group.name }}</p>
              <div class="flex flex-wrap gap-1.5">
                <!-- VNDB's own English vocabulary, as published: the catalog
                     has no Chinese localization for it, and a hand-rolled one
                     here would drift from the source it came from. -->
                <KunChip
                  v-for="trait in group.traits"
                  :key="trait.id"
                  size="xs"
                  :color="trait.spoiler > 0 ? 'warning' : 'default'"
                >
                  {{ trait.name }}<template v-if="trait.lie">（伪）</template>
                </KunChip>
              </div>
            </div>
          </div>

          <KunButton
            v-if="hiddenTraitCount && !isTraitSpoilerRevealed"
            variant="flat"
            color="warning"
            size="sm"
            @click="isTraitSpoilerRevealed = true"
          >
            <KunIcon name="lucide:eye" />
            显示 {{ hiddenTraitCount }} 条剧透特征
          </KunButton>

          <div
            v-if="data.links.length"
            class="flex flex-wrap items-center gap-3"
          >
            <template v-for="link in data.links" :key="link.source">
              <KunLink
                v-if="link.url"
                :to="link.url"
                target="_blank"
                rel="noopener noreferrer"
                size="sm"
                color="default"
                class-name="text-default-500 hover:text-default-700"
              >
                {{ link.name }}
                <KunIcon name="lucide:external-link" class="inline size-3" />
              </KunLink>
              <!-- No verified character-page template for this source. Showing
                   the name without a link beats guessing a URL that 404s. -->
              <span v-else class="text-default-400 text-sm">{{
                link.name
              }}</span>
            </template>
          </div>

          <p class="text-default-500 text-sm">
            资料来自 NextMoe 目录的角色图谱。默认仅显示 SFW 的 Galgame, 查看
            NSFW Galgame 请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>
        </div>
      </template>
    </KunHeader>

    <!-- The site's ordinary galgame card, not a lookalike. What this page adds
         through the #meta slot is the one thing an appearance list needs on
         top: who voiced her in THAT game — a recast between an original and its
         remake is a real event, and a single CV line would erase it. -->
    <GalgameCard v-if="works.length" :galgames="works" :is-transparent="false">
      <template #meta="{ galgame }">
        <p
          v-if="galgame.voices?.length"
          class="text-default-500 mt-1 line-clamp-2 text-xs"
        >
          CV {{ galgame.voices.map((v) => v.name).join(' / ') }}
        </p>
      </template>
    </GalgameCard>

    <KunNull v-else description="暂无该角色登场的 Galgame" />

    <div v-if="nextOffset !== null" class="flex justify-center">
      <KunButton
        variant="flat"
        color="primary"
        :is-loading="loadingMore"
        @click="loadMore"
      >
        加载更多作品
      </KunButton>
    </div>
  </div>
</template>
