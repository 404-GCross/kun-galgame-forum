<script setup lang="ts">
import {
  KUN_GALGAME_CHARACTER_KIND_MAP,
  KUN_GALGAME_CHARACTER_KIND_COLOR,
  KUN_GALGAME_CHARACTER_SPOILER_MAP,
  getGalgameCharacterSourceName
} from '~/constants/galgameCharacter'

// The character popup on a game's 登场角色 panel.
//
// It opens with what the roster already knows — name, billing, CV, both
// artworks — so there is never a spinner in front of an empty box, and then
// fills in what only the character's own record has (简介 and VNDB traits) from
// `works=0`, the identity-alone face. The appearance list is deliberately NOT
// fetched: this popup is "who is she", and "every game she is in" is the whole
// reason the full page exists.
//
// Responses are memoized per character id for the lifetime of the panel —
// reopening the same character on the same page is free, and a reader comparing
// two heroines does it constantly.
const props = defineProps<{
  character: GalgameDetailCharacter | null
}>()

const isOpen = defineModel<boolean>({ required: true })

const cache = new Map<number, GalgameCharacterDetail>()
const detail = ref<GalgameCharacterDetail | null>(null)
const isLoading = ref(false)

const load = async (id: number) => {
  const cached = cache.get(id)
  if (cached) {
    detail.value = cached
    return
  }
  detail.value = null
  isLoading.value = true
  const res = await kunFetch<GalgameCharacterDetail>(
    `/galgame-character/${id}`,
    {
      method: 'GET',
      query: { works: 0 }
    }
  )
  isLoading.value = false
  // A character the registry moved or dropped simply contributes nothing extra
  // here: the roster material above is still true, and the popup keeps showing
  // it rather than turning into an error box.
  if (!res || res.moved_to) {
    return
  }
  cache.set(id, res)
  // The reader may already have clicked on to someone else while this was in
  // flight; a late response must not overwrite the character now on screen.
  if (props.character?.id === id) {
    detail.value = res
  }
}

watch(
  () => [isOpen.value, props.character?.id] as const,
  ([open, id]) => {
    if (open && id) {
      load(id)
    }
  },
  { immediate: true }
)

// Both artworks are shown at ORIGINAL size here, so each is framed by its own
// measured shape and nothing is cropped to fit a box it was never drawn for.
//
// The roster line and the character's own record describe the same two
// pictures, so either may supply the shape: the roster's arrives first and is
// what the popup opens with, and the record's covers the case where the roster
// was served before image_service could size it. Neither is a fallback GUESS —
// when both are silent the frame is simply absent and the picture lays itself
// out.
const figureFrame = computed(() =>
  artFrame(props.character?.figure_meta, detail.value?.figure_meta)
)
const bustFrame = computed(() =>
  artFrame(props.character?.image_meta, detail.value?.image_meta)
)

const kindText = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_KIND_MAP[props.character.kind] || ''
    : ''
)
const kindColor = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_KIND_COLOR[props.character.kind] || 'default'
    : 'default'
)
const spoilerText = computed(() =>
  props.character
    ? KUN_GALGAME_CHARACTER_SPOILER_MAP[props.character.spoiler]
    : ''
)

// Spoiler traits arrive with the response and wait for a click — see the
// backend's spoilers ceiling. `lie` traits stay in the list with their marker:
// "looks true, is not" is a fact about the character, not a data defect.
const isTraitSpoilerRevealed = ref(false)
const traits = computed(() => {
  const all = detail.value?.traits ?? []
  return isTraitSpoilerRevealed.value ? all : all.filter((t) => t.spoiler === 0)
})
const hiddenTraitCount = computed(
  () => (detail.value?.traits ?? []).filter((t) => t.spoiler > 0).length
)

// Grouped, in first-seen order: the catalog already sorts traits so that a
// group's members are contiguous, so nothing here re-sorts either.
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

// Same one line the page uses: where the paragraph came from, and whether a
// machine wrote the translation. The popup shows only the lead bio — the other
// languages are what the full page is for — but it must still credit the one it
// does show.
const introCredit = computed(() => {
  const lead = detail.value?.intros.find((i) => i.intro === detail.value?.intro)
  const parts: string[] = []
  if (lead?.source) {
    parts.push(`简介来自 ${getGalgameCharacterSourceName(lead.source)}`)
  }
  if (lead?.machine) {
    parts.push('由机器翻译生成')
  }
  return parts.join(' · ')
})

// Reset the per-character view state whenever the popup switches subject, so
// one character's revealed spoilers are never already-revealed on the next.
watch(
  () => props.character?.id,
  () => {
    isTraitSpoilerRevealed.value = false
  }
)
</script>

<template>
  <KunModal v-model="isOpen" size="xl" scroll-behavior="inside">
    <div v-if="character" class="space-y-4">
      <div class="flex flex-col gap-4 sm:flex-row">
        <!-- The art column. Here each picture stands alone, so each gets its
             OWN ratio rather than a shared frame — nothing is cropped and
             nothing is letterboxed. When both exist the figure leads and the
             bust rides beside it: a portrait is often a different pose, not a
             crop of the same drawing.
             Both open full-size in the lightbox, the same as on the character
             page: this popup shrinks a 立绘 to thumbnail width, and the art is
             half the reason a reader clicked the character at all. -->
        <KunLightboxGallery v-if="character.figure || character.image">
          <div class="flex shrink-0 gap-3 sm:flex-col">
            <KunLightboxGalleryItem
              v-if="character.figure"
              :src="character.figure"
              :alt="character.name"
              :wrap="false"
              v-slot="{ open }"
            >
              <button
                type="button"
                class="bg-default-100 cursor-zoom-in overflow-hidden rounded-xl"
                :aria-label="`查看 ${character.name} 的立绘`"
                @click="open"
              >
                <KunImage
                  :src="character.figure"
                  :alt="character.name"
                  loading="eager"
                  :aspect-ratio="figureFrame.aspectRatio"
                  :object-fit="figureFrame.objectFit"
                  :thumbhash="figureFrame.thumbhash"
                  class-name="w-40 sm:w-56"
                />
              </button>
            </KunLightboxGalleryItem>

            <KunLightboxGalleryItem
              v-if="character.image"
              :src="character.image"
              :alt="character.name"
              :wrap="false"
              v-slot="{ open }"
            >
              <button
                type="button"
                class="bg-default-100 cursor-zoom-in overflow-hidden rounded-xl"
                :aria-label="`查看 ${character.name} 的头像`"
                @click="open"
              >
                <KunImage
                  :src="character.image"
                  :alt="character.name"
                  loading="eager"
                  :aspect-ratio="bustFrame.aspectRatio"
                  :object-fit="bustFrame.objectFit"
                  :thumbhash="bustFrame.thumbhash"
                  :class-name="
                    character.figure ? 'w-24 sm:w-28' : 'w-40 sm:w-56'
                  "
                />
              </button>
            </KunLightboxGalleryItem>
          </div>
        </KunLightboxGallery>

        <div class="min-w-0 grow space-y-3">
          <div class="space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="text-foreground text-xl font-medium">
                {{ character.name }}
              </h3>
              <KunChip v-if="kindText" :color="kindColor" size="sm">
                {{ kindText }}
              </KunChip>
              <KunChip v-if="spoilerText" color="warning" size="sm">
                {{ spoilerText }}
              </KunChip>
            </div>
            <p
              v-if="character.latin && character.latin !== character.name"
              class="text-default-400 text-sm"
            >
              {{ character.latin }}
            </p>
          </div>

          <div v-if="character.voices.length" class="text-default-500 text-sm">
            CV
            <template v-for="(v, index) in character.voices" :key="v.id">
              <span v-if="index"> / </span>
              <KunLink
                :to="`/galgame/staff/${v.id}`"
                underline="none"
                size="sm"
                class-name="text-default-600 hover:text-primary"
              >
                {{ v.name }}
              </KunLink>
            </template>
          </div>

          <KunLoading v-if="isLoading" />

          <template v-else-if="detail">
            <div v-if="detail.intro" class="space-y-1">
              <p
                class="text-default-600 max-h-48 overflow-y-auto text-sm whitespace-pre-line"
              >
                {{ detail.intro }}
              </p>
              <!-- Said out loud rather than passed off as the site's own: a
                   borrowed, sometimes machine-translated bio is worth reading
                   AND worth knowing about. -->
              <p v-if="introCredit" class="text-default-400 text-xs">
                {{ introCredit }}
              </p>
            </div>

            <div v-if="traitGroups.length" class="space-y-2">
              <div
                v-for="group in traitGroups"
                :key="group.name"
                class="space-y-1"
              >
                <p class="text-default-400 text-xs">{{ group.name }}</p>
                <div class="flex flex-wrap gap-1.5">
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

            <!-- The identity anchors. Cheap, and the reason a reader opens a
                 character popup is often to go and read more elsewhere. A
                 source with no verified template renders as plain text rather
                 than as a guessed URL that 404s. -->
            <div
              v-if="detail.links.length"
              class="flex flex-wrap items-center gap-3"
            >
              <template v-for="link in detail.links" :key="link.source">
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
                <span v-else class="text-default-400 text-sm">
                  {{ link.name }}
                </span>
              </template>
            </div>
          </template>
        </div>
      </div>

      <div class="flex justify-end">
        <KunButton
          :href="`/galgame/character/${character.id}`"
          variant="flat"
          color="primary"
        >
          查看角色详情
          <KunIcon name="lucide:arrow-right" />
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
