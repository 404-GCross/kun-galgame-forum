<script setup lang="ts">
// Favorite is now membership in one or more collections (收藏夹). The heart is a
// button that opens the collection picker; its filled state reflects whether the
// game is in >=1 of the user's collections. targetUserId is accepted (call sites
// still pass it) but unused — the owner economics live entirely on the server.
const props = defineProps<{
  galgameId: number
  targetUserId?: number
  favoriteCount: number
  isFavorited: boolean
}>()

const { id } = usePersistUserStore()

const isFavorited = ref(props.isFavorited)
const favoriteCount = ref(props.favoriteCount)

// The feed hydrates is-favorited asynchronously (see useMyGalgameInteractions),
// so reflect a late-arriving initial state. Harmless on the detail page.
watch(
  () => props.isFavorited,
  (value) => (isFavorited.value = value)
)
watch(
  () => props.favoriteCount,
  (value) => (favoriteCount.value = value)
)

const pickerOpen = ref(false)

const onClick = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  pickerOpen.value = true
}

const onSaved = (payload: { favorited: boolean }) => {
  if (isFavorited.value !== payload.favorited) {
    favoriteCount.value += payload.favorited ? 1 : -1
  }
  isFavorited.value = payload.favorited
  // Keep the shared feed state in sync so other cards reflect the change.
  useMyGalgameInteractions().setFavorited(props.galgameId, payload.favorited)
}
</script>

<template>
  <KunTooltip text="收藏">
    <!-- Sized to match KunReaction's default `md` (used by the like button):
         gap-1.5 px-2 py-1 text-sm, icon 1.15rem. -->
    <button
      type="button"
      aria-label="收藏"
      :class="
        cn(
          'relative inline-flex cursor-pointer select-none items-center gap-1.5 rounded-full px-2 py-1 text-sm transition-colors',
          isFavorited
            ? 'text-danger hover:bg-default-100/60'
            : 'text-default-500 hover:bg-default-100'
        )
      "
      @click="onClick"
    >
      <KunIcon
        name="lucide:heart"
        :class="cn('text-[1.15rem] transition-colors', isFavorited && 'fill-current')"
      />
      <span v-if="favoriteCount">{{ favoriteCount }}</span>
    </button>
  </KunTooltip>

  <GalgameCollectionPickerModal
    v-model="pickerOpen"
    :galgame-id="galgameId"
    @saved="onSaved"
  />
</template>
