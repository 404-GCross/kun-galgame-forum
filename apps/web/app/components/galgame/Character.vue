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
// The panel is deliberately TWO surfaces rather than one grid:
//
//   · a portrait grid for the characters that have art. Coverage is partial and
//     always will be — the registry only has pictures for the cast the upstream
//     sources bothered to publish — so a single grid would have been mostly
//     blank tiles.
//   · a plain name list for the rest. A character with no picture is still part
//     of the cast, and its 声优 is the same fact it always was.
//
// Clicking a portrait opens the FULL-SIZE art in the shared lightbox, which is
// where a 立绘 finally gets the room it needs.
const props = defineProps<{
  characters: GalgameDetailCharacter[]
}>()

// The full-body 立绘 wins over the bust when a character has both — it is the
// richer asset and the reason this panel shows pictures at all. The two are
// DIFFERENT ASSETS, not two sizes of one, which is why the fit differs with
// them: the bust was cover-cropped to 256×360 upstream and belongs in a
// portrait box, while the figure is a whole person standing on a white field
// and must keep its own ratio. Crop a figure into a 3:4 box and what is left is
// a midriff.
const artOf = (c: GalgameDetailCharacter) => c.figure || c.image || ''
const isFigure = (c: GalgameDetailCharacter) => !!c.figure

// Tiles load the `mini` variant (both catalog presets generate one); the
// lightbox opens the original.
const thumbOf = (c: GalgameDetailCharacter) =>
  withImageVariant(artOf(c), 'mini')

// A spoiler here is the character's PRESENCE, so blurring the picture would not
// help — the name gives it away just as fast. They are withheld from both lists
// entirely and revealed together, on one explicit click. Only the VNDB lane
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

const portraits = computed(() => visible.value.filter((c) => !!artOf(c)))
const nameOnly = computed(() => visible.value.filter((c) => !artOf(c)))

// Two rows of the widest grid. A long-running series' roster runs to dozens of
// characters, and the panel sits above the screenshots — it must not become the
// page.
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
</script>

<template>
  <div v-if="characters.length" class="space-y-3">
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

    <KunLightboxGallery v-if="visiblePortraits.length">
      <!-- auto-fill so the tiles keep a sane size in the 2/3 content column and
           still reflow to 3 across on a phone. -->
      <div
        class="grid grid-cols-3 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(140px,1fr))]"
      >
        <div v-for="c in visiblePortraits" :key="c.id" class="space-y-1.5">
          <KunLightboxGalleryItem
            :src="artOf(c)"
            :alt="c.name"
            :wrap="false"
            v-slot="{ open }"
          >
            <!-- The tile sits on its own light surface for the same reason the
                 会社 logo does: a 立绘 is cut out against white, and a contained
                 image leaves bars that need a background to be bars OF. -->
            <button
              type="button"
              class="group bg-default-100 hover:ring-primary focus:ring-primary relative block w-full cursor-zoom-in overflow-hidden rounded-lg ring-1 ring-transparent transition-all focus:outline-none"
              :aria-label="`查看 ${c.name} 的立绘`"
              @click="open"
            >
              <KunImage
                :src="thumbOf(c)"
                :alt="c.name"
                loading="lazy"
                aspect-ratio="3/4"
                :object-fit="isFigure(c) ? 'contain' : 'cover'"
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

              <!-- Says what the click is worth: a full standing figure rather
                   than a bigger head. -->
              <span
                v-if="isFigure(c)"
                class="absolute right-1 bottom-1 rounded bg-black/55 px-1.5 py-0.5 text-[0.625rem] text-white"
              >
                立绘
              </span>
            </button>
          </KunLightboxGalleryItem>

          <div class="space-y-0.5">
            <p class="text-default-800 truncate text-sm font-medium">
              {{ c.name }}
            </p>
            <p
              v-if="c.latin && c.latin !== c.name"
              class="text-default-400 truncate text-xs"
            >
              {{ c.latin }}
            </p>
            <p
              v-if="c.voices.length"
              class="text-default-500 truncate text-xs"
              :title="c.voices.map((v) => v.name).join(' / ')"
            >
              CV
              <template v-for="(v, index) in c.voices" :key="v.id">
                <span v-if="index"> / </span>
                <KunLink
                  :to="`/galgame/staff/${v.id}`"
                  underline="none"
                  size="sm"
                  class-name="text-default-500 hover:text-primary"
                >
                  {{ v.name }}
                </KunLink>
              </template>
            </p>
            <p
              v-if="KUN_GALGAME_CHARACTER_SPOILER_MAP[c.spoiler]"
              class="text-warning text-xs"
            >
              {{ KUN_GALGAME_CHARACTER_SPOILER_MAP[c.spoiler] }}
            </p>
          </div>
        </div>
      </div>
    </KunLightboxGallery>

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
      <p v-if="visiblePortraits.length" class="text-default-500 text-sm">
        其他登场角色
      </p>
      <div class="flex flex-wrap items-baseline gap-x-4 gap-y-1.5">
        <span v-for="c in nameOnly" :key="c.id" class="text-sm">
          <span class="text-default-800">{{ c.name }}</span>
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
  </div>
</template>
