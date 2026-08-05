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
           all rather than an empty one. -->
      <template v-if="data.photo" #headerEndContent>
        <!-- Full size, like the 会社 logo in the same slot: the frame is large
             enough for a downscaled variant to show its resampling, and the
             catalog scope promises no particular preset for a portrait. -->
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

    <!-- The site's ordinary galgame card, not a lookalike: same cover framing,
         same badges, same reader settings — and for the works the forum has
         ingested, the same view / like / platform data. What this page adds is
         the only thing a filmography needs on top, through the card's #meta
         slot: what this person did on each game. -->
    <GalgameCard v-if="works.length" :galgames="works" :is-transparent="false">
      <template #meta="{ galgame }">
        <div class="mt-2 flex flex-wrap gap-1">
          <KunChip v-for="role in galgame.roles" :key="role" size="xs">
            {{ role }}
          </KunChip>
        </div>

        <!-- For a voice actor the cast IS the credit: a bare 声优 chip says
             nothing a reader on this page did not already know. -->
        <p
          v-if="galgame.characters?.length"
          class="text-default-500 mt-1 line-clamp-2 text-xs"
        >
          {{ galgame.characters.join(' / ') }}
        </p>
      </template>
    </GalgameCard>

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
