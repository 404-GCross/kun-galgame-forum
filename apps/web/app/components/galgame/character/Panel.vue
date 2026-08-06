<script setup lang="ts">
import {
  KUN_GALGAME_CHARACTER_KIND_MAP,
  KUN_GALGAME_CHARACTER_KIND_COLOR,
  KUN_GALGAME_CHARACTER_SPOILER_MAP
} from '~/constants/galgameCharacter'

// 登场角色: the game's cast, from the catalog's character roster. The backend
// hands it over already merged (appearance edges ∪ voice credits) and already
// ordered (主角 → 配角 → 登场 → 未分类, then by name), so nothing here re-sorts.
//
// THREE surfaces, because the registry gives three different amounts of
// material and one grid would have to lie about that:
//
//   · a large card for each character with a 全身立绘. Those arrive from one
//     source only — Getchu, whose product pages give standing art to the
//     characters the box is selling — so in practice this row is the leads.
//     That correlation is why they get the space; it is NOT stated as a fact,
//     because the catalog has no such field and the reader would take a 攻略
//     label as one.
//   · a dense BUST grid for everyone else with a picture. Busts are cropped
//     3:4 upstream and a figure is not, so mixing the two in one grid is what
//     made the old panel look broken — here each shape gets its own surface.
//   · a plain name list for the rest. A character with no picture is still
//     part of the cast, and her 声优 is the same fact it always was.
//
// Clicking anyone — tile, card or name — opens the same modal.
const props = defineProps<{
  characters: GalgameDetailCharacter[]
}>()

// A spoiler here is the character's PRESENCE, so blurring the picture would not
// help — the name gives it away just as fast. They are withheld from all three
// surfaces and revealed together, on one explicit click. Only the VNDB lane
// populates the level, so 0 means "nobody flagged this" rather than "safe".
const isSpoilerRevealed = ref(false)
const visible = computed(() =>
  isSpoilerRevealed.value
    ? props.characters
    : props.characters.filter((c) => c.spoiler === 0)
)
const spoilerCount = computed(
  () => props.characters.filter((c) => c.spoiler > 0).length
)

const featured = computed(() => visible.value.filter((c) => !!c.figure))
const portraits = computed(() =>
  visible.value.filter((c) => !c.figure && !!c.image)
)
const nameOnly = computed(() =>
  visible.value.filter((c) => !c.figure && !c.image)
)

// Tiles load the `mini` variant (both catalog presets generate one). The bust
// preset COVER-crops to 256×360, so a bust tile is safely 3:4 whatever the
// original was; the figure preset fits INSIDE 360×360, so a figure keeps the
// original's ratio and the frame has to be told what that is.
const thumbOf = (url: string) => withImageVariant(url, 'mini')

// One ratio for the whole 立绘 row rather than one per card — see artGridRatio.
// Square is the fallback because it is the shape most Getchu standing art
// arrives in, and it is only reached when image_service answered for none of
// them.
const figureRatio = computed(() =>
  artGridRatio(
    featured.value.map((c) => c.figure_meta),
    '1/1'
  )
)

// Two rows of the widest bust grid. A long-running series' roster runs to
// dozens of characters, and the panel sits above the screenshots — it must not
// become the page. The figure cards are never collapsed: there are rarely more
// than a handful, and they are the reason to look at this panel at all.
const COLLAPSED_PORTRAITS = 12
const isExpanded = ref(false)
const isCollapsible = computed(
  () => portraits.value.length > COLLAPSED_PORTRAITS
)
const visiblePortraits = computed(() =>
  isCollapsible.value && !isExpanded.value
    ? portraits.value.slice(0, COLLAPSED_PORTRAITS)
    : portraits.value
)

const kindText = (kind: string) => KUN_GALGAME_CHARACTER_KIND_MAP[kind] || ''
// 配角 / 登场 are ordinary rows; only the main cast is coloured. An unmapped
// billing (the catalog's `unknown`, and anything a future wave adds) draws no
// badge at all rather than a 未知 chip nobody asked for.
const kindColor = (kind: string) =>
  KUN_GALGAME_CHARACTER_KIND_COLOR[kind] || 'default'

const activeCharacter = ref<GalgameDetailCharacter | null>(null)
const isModalOpen = ref(false)
const open = (character: GalgameDetailCharacter) => {
  activeCharacter.value = character
  isModalOpen.value = true
}
</script>

<template>
  <div v-if="characters.length" class="space-y-4">
    <div class="flex flex-wrap items-end justify-between gap-2">
      <KunHeader
        name="登场角色"
        description="该 Galgame 的登场角色与配音演员, 数据来自 NextMoe 目录的角色图谱"
        scale="h2"
      />

      <KunButton
        v-if="spoilerCount"
        variant="flat"
        color="warning"
        size="sm"
        @click="isSpoilerRevealed = !isSpoilerRevealed"
      >
        <KunIcon :name="isSpoilerRevealed ? 'lucide:eye-off' : 'lucide:eye'" />
        {{
          isSpoilerRevealed
            ? `隐藏 ${spoilerCount} 名剧透角色`
            : `显示 ${spoilerCount} 名剧透角色`
        }}
      </KunButton>
    </div>

    <!-- The 立绘 cards. A standing figure is a tall-ish drawing on a flat
         field and arrives at whatever ratio the source drew it, so the frame is
         square and the art is CONTAINED in it: the picture keeps its own shape
         and the leftover is quiet background rather than a cropped torso. -->
    <div
      v-if="featured.length"
      class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4"
    >
      <button
        v-for="c in featured"
        :key="c.id"
        type="button"
        class="group bg-default-100 hover:ring-primary focus:ring-primary flex flex-col overflow-hidden rounded-xl text-left ring-1 ring-transparent transition-all focus:outline-none"
        :aria-label="`查看角色 ${c.name}`"
        @click="open(c)"
      >
        <div class="relative w-full">
          <KunImage
            :src="thumbOf(c.figure!)"
            :alt="c.name"
            loading="lazy"
            :aspect-ratio="figureRatio"
            :thumbhash="c.figure_meta?.thumbhash"
            object-fit="contain"
            class-name="w-full"
            image-class-name="transition-transform duration-200 group-hover:scale-105"
          />

          <KunChip
            v-if="kindText(c.kind)"
            :color="kindColor(c.kind)"
            size="xs"
            class-name="absolute top-2 left-2"
          >
            {{ kindText(c.kind) }}
          </KunChip>
        </div>

        <div class="bg-default-50 w-full space-y-0.5 px-3 py-2">
          <p class="text-default-800 truncate font-medium">{{ c.name }}</p>
          <p
            v-if="c.latin && c.latin !== c.name"
            class="text-default-400 truncate text-xs"
          >
            {{ c.latin }}
          </p>
          <p v-if="c.voices.length" class="text-default-500 truncate text-xs">
            CV {{ c.voices.map((v) => v.name).join(' / ') }}
          </p>
          <p
            v-if="KUN_GALGAME_CHARACTER_SPOILER_MAP[c.spoiler]"
            class="text-warning text-xs"
          >
            {{ KUN_GALGAME_CHARACTER_SPOILER_MAP[c.spoiler] }}
          </p>
        </div>
      </button>
    </div>

    <!-- Everyone else with a picture. Busts only, so every frame here is the
         same 3:4 and can be safely cover-cropped. -->
    <div
      v-if="visiblePortraits.length"
      class="grid grid-cols-3 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(120px,1fr))]"
    >
      <button
        v-for="c in visiblePortraits"
        :key="c.id"
        type="button"
        class="group space-y-1.5 text-left"
        :aria-label="`查看角色 ${c.name}`"
        @click="open(c)"
      >
        <div
          class="bg-default-100 group-hover:ring-primary group-focus:ring-primary relative overflow-hidden rounded-lg ring-1 ring-transparent transition-all"
        >
          <KunImage
            :src="thumbOf(c.image!)"
            :alt="c.name"
            loading="lazy"
            aspect-ratio="3/4"
            :thumbhash="c.image_meta?.thumbhash"
            object-fit="cover"
            class-name="w-full"
            image-class-name="transition-transform duration-200 group-hover:scale-105"
          />

          <KunChip
            v-if="kindText(c.kind)"
            :color="kindColor(c.kind)"
            size="xs"
            class-name="absolute top-1 left-1"
          >
            {{ kindText(c.kind) }}
          </KunChip>
        </div>

        <div class="space-y-0.5">
          <p class="text-default-800 truncate text-sm font-medium">
            {{ c.name }}
          </p>
          <p
            v-if="c.voices.length"
            class="text-default-500 truncate text-xs"
            :title="c.voices.map((v) => v.name).join(' / ')"
          >
            CV {{ c.voices.map((v) => v.name).join(' / ') }}
          </p>
          <p
            v-if="KUN_GALGAME_CHARACTER_SPOILER_MAP[c.spoiler]"
            class="text-warning text-xs"
          >
            {{ KUN_GALGAME_CHARACTER_SPOILER_MAP[c.spoiler] }}
          </p>
        </div>
      </button>
    </div>

    <KunButton
      v-if="isCollapsible"
      variant="flat"
      color="primary"
      size="sm"
      @click="isExpanded = !isExpanded"
    >
      <KunIcon
        :name="isExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
      />
      {{
        isExpanded
          ? '收起角色'
          : `展开其余 ${portraits.length - COLLAPSED_PORTRAITS} 名角色`
      }}
    </KunButton>

    <!-- The cast the registry has no picture for. Same rows, no tiles — the
         alternative was a grid of empty frames, which says "missing" far louder
         than it says "character". -->
    <div v-if="nameOnly.length" class="space-y-1.5">
      <p
        v-if="featured.length || visiblePortraits.length"
        class="text-default-500 text-sm"
      >
        其他登场角色
      </p>
      <div class="flex flex-wrap items-baseline gap-x-4 gap-y-1.5">
        <span v-for="c in nameOnly" :key="c.id" class="text-sm">
          <button
            type="button"
            class="text-default-800 hover:text-primary cursor-pointer"
            @click="open(c)"
          >
            {{ c.name }}
          </button>
          <span v-if="c.voices.length" class="text-default-400">
            （CV
            <template v-for="(v, index) in c.voices" :key="v.id">
              <span v-if="index"> / </span>
              <KunLink
                :to="`/galgame/staff/${v.id}`"
                underline="none"
                size="sm"
                class-name="text-default-400 hover:text-primary"
              >
                {{ v.name }}
              </KunLink>
            </template>
            ）
          </span>
        </span>
      </div>
    </div>

    <GalgameCharacterModal v-model="isModalOpen" :character="activeCharacter" />
  </div>
</template>
