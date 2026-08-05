<script setup lang="ts">
import { KUN_GALGAME_STAFF_GENDER_MAP } from '~/constants/galgameStaff'

// `id` is a CATALOG CREDIT-NAME id — the id the 制作人员 panel on every game
// detail page already carries, so the link needs no lookup.
//
// The page describes a NAME, and the header says so where it matters: a name
// with published siblings shows them as "同一人的其他名义", and one without
// simply does not claim to be the whole person. Guessing the link from spellings
// is the one thing this page must not do — the registry's person resolution is
// evidence-based and its visibility policy is deliberate.
const route = useRoute()
const staffId = computed(() => Number((route.params as { id: string }).id))

if (!Number.isInteger(staffId.value) || staffId.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该制作人员',
    fatal: true
  })
}

const PAGE_SIZE = 50

const { data } = await useKunFetch<GalgameStaffDetail>(
  `/galgame-staff/${staffId.value}`,
  { method: 'GET', query: { limit: PAGE_SIZE }, watch: false }
)

if (!data.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该制作人员',
    fatal: true
  })
}

// The filmography is offset-paged with no total, so this is a 加载更多 rather
// than a pager: we know whether ANOTHER page exists, never how many.
const works = ref<GalgameStaffWork[]>([...data.value.works])
const nextOffset = ref<number | null>(data.value.next_offset)
const loadingMore = ref(false)

const loadMore = async () => {
  if (nextOffset.value === null || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const res = await kunFetch<GalgameStaffDetail>(
    `/galgame-staff/${staffId.value}`,
    { method: 'GET', query: { limit: PAGE_SIZE, offset: nextOffset.value } }
  )
  loadingMore.value = false
  if (!res) {
    return
  }
  works.value.push(...res.works)
  nextOffset.value = res.next_offset
}

// Only the games this forum has an entry for can be opened. The rest still
// render — a career is not the subset we happen to host — but as plain cards.
const workHref = (work: GalgameStaffWork) =>
  work.id > 0 ? `/galgame/${work.id}` : undefined

const releaseYear = (date: string | null) => date?.slice(0, 4) ?? ''

// The person's own facts, which the registry publishes only where the
// name→person link is public — a hidden link arrives zeroed, and each row is
// then simply absent. Every field is optional on the wire as well: this page
// renders against a catalog that may not ship them yet, and 「未知」 rows would
// be the only thing such a reader ever saw.
const genderText = computed(() =>
  data.value?.gender ? KUN_GALGAME_STAFF_GENDER_MAP[data.value.gender] : ''
)

// Fuzzy by design — a year alone and a month+day with no year are both whole
// answers here, so the formatter renders from the parts rather than a Date.
const birthdayText = computed(() =>
  formatFuzzyDate(data.value?.birth_y, data.value?.birth_m, data.value?.birth_d)
)

const subtitle = computed(() => {
  const parts = [data.value?.name_zh, data.value?.latin].filter(
    (part): part is string => !!part && part !== data.value?.name
  )
  return parts.join(' · ')
})

useKunSeoMeta({
  title: `${data.value.name} 参与制作的 Galgame`,
  description:
    data.value.intro ||
    `${data.value.name} 在本站收录的 Galgame 中担任 ${data.value.roles.join(' / ')} 等职位的作品一览。`
})
</script>

<template>
  <div v-if="data" class="space-y-6">
    <KunHeader :name="data.name" :description="subtitle">
      <!-- The portrait describes the PERSON, and the registry publishes it only
           where the name→person link is public — so a name with none looks
           exactly like a name whose link is hidden. Both render as no frame at
           all rather than an empty one.
           Full size, like the 会社 logo in the same slot: the frame is large
           enough for a downscaled variant to show its resampling, and the
           catalog scope promises no particular preset for a portrait. -->
      <template v-if="data.photo" #headerEndContent>
        <KunImage
          :src="data.photo"
          :alt="data.name"
          loading="eager"
          object-fit="cover"
          class-name="w-28 shrink-0 rounded-lg sm:w-32"
          :style="{ aspectRatio: '3/4' }"
        />
      </template>

      <template #endContent>
        <div class="space-y-3">
          <!-- Each row appears only where the registry published that fact.
               Nothing here says 「未知」: a name whose person link is private is
               indistinguishable from one the registry has no person for, and
               the page does not pretend to know which it is looking at. -->
          <div
            v-if="genderText || birthdayText"
            class="text-default-500 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm"
          >
            <span v-if="genderText">性别 {{ genderText }}</span>
            <span v-if="birthdayText">生日 {{ birthdayText }}</span>
          </div>

          <p v-if="data.intro" class="text-default-600">{{ data.intro }}</p>

          <div v-if="data.roles.length" class="flex flex-wrap gap-2">
            <!-- Roles carry no count on purpose: the filmography is paged and
                 the catalog publishes no total, so a number here could only
                 ever describe the page that happens to be loaded. -->
            <KunChip v-for="role in data.roles" :key="role" color="primary">
              {{ role }}
            </KunChip>
          </div>

          <div v-if="data.siblings.length" class="space-y-1">
            <p class="text-default-500 text-sm">同一人的其他名义</p>
            <div class="flex flex-wrap gap-x-4 gap-y-1">
              <KunLink
                v-for="sibling in data.siblings"
                :key="sibling.id"
                :to="`/galgame/staff/${sibling.id}`"
                underline="none"
                class-name="text-foreground hover:text-primary font-medium"
              >
                {{ sibling.name }}
              </KunLink>
            </div>
          </div>

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
              <!-- No verified person-page template for this source. Showing the
                   name without a link beats guessing a URL that 404s. -->
              <span v-else class="text-default-400 text-sm">{{
                link.name
              }}</span>
            </template>
          </div>

          <p class="text-default-500 text-sm">
            资料来自 NextMoe 目录的署名图谱。本页是一个「署名名义」而非人物档案,
            同一位创作者可能以多个名义署名。默认仅显示 SFW 的 Galgame, 查看 NSFW
            Galgame 请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>
        </div>
      </template>
    </KunHeader>

    <div
      v-if="works.length"
      class="grid grid-cols-2 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
    >
      <KunCard
        v-for="work in works"
        :key="work.catalog_id"
        :is-transparent="false"
        :href="workHref(work)"
        class-name="h-full p-0"
        content-class="flex h-full flex-col gap-2 p-3"
      >
        <template #cover>
          <KunImage
            :src="withBannerVariant(work.banner, 'mini')"
            loading="lazy"
            :alt="getPreferredLanguageText(work.name)"
            placeholder="/placeholder.webp"
            :thumbhash="work.banner_thumbhash"
            class="w-full object-cover"
            :style="{ aspectRatio: '16/9' }"
          />
        </template>

        <p class="line-clamp-2 font-medium">
          {{ getPreferredLanguageText(work.name) }}
        </p>

        <div class="flex flex-wrap gap-1">
          <KunChip v-for="role in work.roles" :key="role" size="xs">
            {{ role }}
          </KunChip>
        </div>

        <!-- For a voice actor the cast IS the credit: a bare 声优 chip says
             nothing a reader on this page did not already know. -->
        <p
          v-if="work.characters?.length"
          class="text-default-500 line-clamp-2 text-xs"
        >
          {{ work.characters.join(' / ') }}
        </p>

        <!-- Pinned to the bottom of the card. The chip row wraps to one or two
             lines depending on how many hats this person wore, and without
             mt-auto the footer rode up with it, leaving the grid ragged. -->
        <div
          class="text-default-400 mt-auto flex items-center justify-between text-xs"
        >
          <!-- A game the forum has no entry for: the credit is real, the page
               is not, so the card says why it does not open. -->
          <span>{{ work.id === 0 ? '本站未收录' : '' }}</span>
          <span>{{ releaseYear(work.release_date) }}</span>
        </div>
      </KunCard>
    </div>

    <KunNull v-else description="暂无该制作人员参与的 Galgame" />

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
