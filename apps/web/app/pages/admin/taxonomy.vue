<script setup lang="ts">
// This staff console is admin-only (admin ⊂ ren): the `admin` middleware
// bounces non-admins. (Editing / deleting taxonomy is also API-gated by
// RequireAdmin; creating stays open for the public contribution surfaces.)
definePageMeta({ middleware: 'admin' })
useKunDisableSeo('Wiki 条目管理')
</script>

<template>
  <AdminTaxonomyContainer />
</template>
