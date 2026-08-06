<script setup lang="ts">
import {
  KUN_GALGAME_CHARACTER_KIND_MAP,
  KUN_GALGAME_CHARACTER_KIND_COLOR,
  KUN_GALGAME_CHARACTER_SPOILER_MAP
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
        <!-- The art column. A 立绘 is CONTAINED and a bust is cropped, for the
             same reason the panel keeps them on separate surfaces: they are
             different assets, and neither is a resize of the other. When both
             exist the figure leads and the bust rides beside it — a portrait
             is often a different pose, not a crop of the same drawing. -->
        <div
          v-if="character.figure || character.image"
          class="flex shrink-0 gap-3 sm:flex-col"
        >
          <div
            v-if="character.figure"
            class="bg-default-100 overflow-hidden rounded-xl"
          >
            <KunImage
              :src="character.figure"
              :alt="character.name"
              loading="eager"
              aspect-ratio="1/1"
              object-fit="contain"
              class-name="w-40 sm:w-56"
            />
          </div>

          <div
            v-if="character.image"
            class="bg-default-100 overflow-hidden rounded-xl"
          >
            <KunImage
              :src="character.image"
              :alt="character.name"
              loading="eager"
              aspect-ratio="3/4"
              object-fit="cover"
              :class-name="character.figure ? 'w-24 sm:w-28' : 'w-40 sm:w-56'"
            />
          </div>
        </div>

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
              <!-- Said out loud rather than passed off as an original: a
                   machine-translated bio is worth reading AND worth knowing
                   about. -->
              <p
                v-if="
                  detail.intros.find((i) => i.intro === detail!.intro)?.machine
                "
                class="text-default-400 text-xs"
              >
                本段简介由机器翻译生成
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
