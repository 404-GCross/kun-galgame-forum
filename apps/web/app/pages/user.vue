<script setup lang="ts">
// Key by the :id param so navigating between two profiles client-side
// (e.g. someone else's → your own via the user menu) REMOUNTS this page —
// otherwise the parent route component is reused and the one-shot setup fetch
// below keeps showing the previous user until a hard refresh. Keyed on id (not
// path) so switching subpages of the same user (info → topic → …) doesn't
// remount needlessly.
definePageMeta({ key: (route) => (route.params as { id: string }).id })

const route = useRoute()

const userId = computed(() => {
  return parseInt((route.params as { id: string }).id)
})

const { data } = await useKunFetch<UserInfo>(`/user/${userId.value}`)

// Banned profiles get a stripped {id, name, status: 1} payload from
// the BE — there's no `'banned'` sentinel string, so the previous
// `data === 'banned'` branch was dead. status !== 0 is the canonical
// "not in good standing" signal.
const isBanned = computed(() => data.value && data.value.status !== 0)

if (isBanned.value) {
  useKunDisableSeo('该用户已被封禁')
} else if (data.value) {
  useKunSeoMeta({
    title: data.value.name,
    description: data.value.bio
  })
} else {
  useKunDisableSeo('未找到该用户')
}
</script>

<template>
  <div>
    <!-- Single REAL root box — NOT `display: contents`. This div is the
         page-transition root: `display: contents` produces no box, so Nuxt sees
         no single root node ("does not have a single root node" warning) and the
         enter animation can't attach — the page snaps in the instant the leave
         finishes (the "teleport" at the tail of the transition). Keep the
         comment INSIDE the root, never before it — a leading comment is itself a
         second root node and re-triggers the same warning. -->
    <div v-if="!isBanned" class="h-[calc(100dvh-120px)]">
      <div v-if="data" class="flex h-full w-full">
        <UserNavBar
          :user="{ id: data.id, name: data.name, avatar: data.avatar }"
        />

        <div class="scrollbar-hide h-full w-full overflow-y-auto pl-3">
          <NuxtPage :user="data" />
        </div>
      </div>

      <KunNull v-else description="未找到该用户" />
    </div>

    <div v-else class="h-[calc(100dvh-120px)]">
      <KunNull description="此用户已被封禁" />
    </div>
  </div>
</template>
