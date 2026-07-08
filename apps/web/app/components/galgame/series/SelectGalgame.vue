<script setup lang="ts">
// Galgame picker for the series-management UI. Search is the shared
// GalgameSearchAutocomplete (KunAutocomplete + the accurate wiki Meilisearch +
// the stale-response guard) — replacing the old hand-rolled dropdown that hit
// the weaker /series/search. v-model is the selected galgame id array; chips
// show the picks, and edit-mode ids are resolved to names via
// /galgame-series/modal.
const props = defineProps<{
  modelValue: number[]
  label?: string
  required?: boolean
}>()

const emits = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

type SelectedGalgame = { id: number; name: string }

const selectedGalgames = ref<SelectedGalgame[]>([])

const internalIds = computed({
  get: () => props.modelValue,
  set: (newValue) => emits('update:modelValue', newValue)
})

const onPick = (hit: { id: number; name: string }) => {
  if (internalIds.value.includes(hit.id)) return
  selectedGalgames.value.push({ id: hit.id, name: hit.name })
  internalIds.value = [...internalIds.value, hit.id]
}

const removeGame = (gameId: number) => {
  selectedGalgames.value = selectedGalgames.value.filter((g) => g.id !== gameId)
  internalIds.value = internalIds.value.filter((id) => id !== gameId)
}

// Edit mode: resolve pre-linked ids to display names (a search pick already
// carries its own name; this only runs when modelValue arrives without matching
// local chips — e.g. opening an existing series).
const syncSelectedGalgames = async (ids: number[]) => {
  if (!ids || ids.length === 0) {
    selectedGalgames.value = []
    return
  }
  const currentIds = selectedGalgames.value.map((g) => g.id)
  if (
    JSON.stringify([...currentIds].sort()) === JSON.stringify([...ids].sort())
  ) {
    return
  }
  const data = await kunFetch<GalgameSeriesSearchItem[]>(
    '/galgame-series/modal',
    { method: 'POST', body: { ids } }
  )
  selectedGalgames.value = (data ?? []).map((g) => ({
    id: g.id,
    name: galgameNameFromWire(g, `#${g.id}`)
  }))
}

watch(() => props.modelValue, syncSelectedGalgames, { immediate: true })
</script>

<template>
  <div>
    <label v-if="label" class="text-default-700 mb-1 block text-sm font-medium">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>

    <div class="space-y-2">
      <!-- selected chips -->
      <div v-if="selectedGalgames.length" class="flex flex-wrap gap-2">
        <span
          v-for="game in selectedGalgames"
          :key="game.id"
          class="bg-primary-100 text-primary-800 flex items-center gap-1.5 rounded px-2 py-0.5 text-sm"
        >
          {{ game.name }}
          <button
            type="button"
            class="text-primary-600 hover:text-primary-900 font-bold"
            @click.stop="removeGame(game.id)"
          >
            ×
          </button>
        </span>
      </div>

      <GalgameSearchAutocomplete
        :exclude-ids="internalIds"
        placeholder="搜索并添加 Galgame..."
        @select="onPick"
      />
    </div>
  </div>
</template>
