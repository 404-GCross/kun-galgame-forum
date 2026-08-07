<script setup lang="ts">
// The indented family tree, recursive through its own auto-import name.
//
// Indentation is deliberately small and identical at every depth (12px, 16px
// from `sm`): the graph runs up to four levels deep, and a phone-width column
// that loses 32px per level has nothing left for the brand names by level
// three. The connector is a single left border rather than per-row elbows —
// same reading, no absolutely-positioned pseudo-elements to keep aligned when
// a row wraps.
defineProps<{
  nodes: GalgameOfficialFamilyNode[]
  currentId: number
}>()
</script>

<template>
  <ul class="space-y-2">
    <li v-for="item in nodes" :key="item.official.id" class="min-w-0 space-y-2">
      <GalgameOfficialRelationNode
        :official="item.official"
        :role="item.role"
        :is-current="item.official.id === currentId"
      />
      <GalgameOfficialRelationTree
        v-if="item.children.length"
        :nodes="item.children"
        :current-id="currentId"
        class="border-default-200 ml-3 border-l pl-3 sm:ml-4 sm:pl-4"
      />
    </li>
  </ul>
</template>
