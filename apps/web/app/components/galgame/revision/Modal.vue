<script setup lang="ts">
// Button + modal wrapper around GalgameRevisionList. Mounted next to the
// edit/delete actions on the taxonomy detail pages (tag / official / engine)
// so the edit history is reachable on demand instead of always rendered at
// the page bottom. Viewing history is public (the revision GETs are public);
// reverting stays gated by `canRevert` (canAdminister), enforced by the list.
import type { RevisionEntity } from '~/composables/useRevisionHistory'

defineProps<{
  entity: RevisionEntity
  id: number
  entityLabel?: string
  canRevert?: boolean
}>()

// The wiki gates taxonomy revision history behind auth, so the button is
// only meaningful for logged-in users (anonymous would hit a 205 → login).
const { id: currentUserId } = usePersistUserStore()

const isOpen = ref(false)
</script>

<template>
  <KunButton
    v-if="currentUserId"
    variant="flat"
    color="default"
    @click="isOpen = true"
  >
    <KunIcon name="lucide:history" />
    编辑历史
  </KunButton>

  <KunModal v-model="isOpen" inner-class-name="max-w-3xl w-full">
    <div class="max-h-[70dvh] overflow-y-auto p-1">
      <GalgameRevisionList
        v-if="isOpen"
        :entity="entity"
        :id="id"
        :entity-label="entityLabel"
        :can-revert="canRevert"
      />
    </div>
  </KunModal>
</template>
