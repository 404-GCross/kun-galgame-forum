<script setup lang="ts">
// Thin wrapper that delegates to the universal GalgameRevisionList
// (K-PR5). The hard work of fetching list + diff + revert is in
// useRevisionHistory; this file only contributes the galgame-specific
// permission gate (creator OR admin/moderator) and the entity wiring.
//
// Pre-K-PR5 this file held all the fetch + render logic inline; that
// has moved to revision/List.vue so the 4 taxonomy detail pages can
// reuse the same UI.
const route = useRoute()
const gid = computed(() => parseInt((route.params as { gid: string }).gid))

const galgame = inject<GalgameDetail>('galgame')
const { id: currentUserId } = usePersistUserStore()
const { canModerate } = useRole()
const canRevert = computed(
  () => galgame?.user.id === currentUserId || canModerate.value
)
</script>

<template>
  <GalgameRevisionList
    entity="galgame"
    :id="gid"
    entity-label="该 Galgame"
    :can-revert="canRevert"
  />
</template>
