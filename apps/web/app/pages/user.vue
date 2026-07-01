<script setup lang="ts">
// Key by the :id param so navigating between two profiles client-side
// (e.g. someone else's → your own via the user menu) REMOUNTS this page —
// otherwise the parent route component is reused and the one-shot setup fetch
// below keeps showing the previous user until a hard refresh. Keyed on id (not
// path) so switching subpages of the same user (info → topic → …) doesn't
// remount needlessly.
import { kunUserMainNav } from '~/constants/user'

definePageMeta({ key: (route) => (route.params as { id: string }).id })

const route = useRoute()

const userId = computed(() => {
  return parseInt((route.params as { id: string }).id)
})

const { data } = await useKunFetch<UserInfo>(`/user/${userId.value}`)

// Owner = the logged-in viewer is on their OWN profile (id match, never a role);
// gates the owner-only 设置 tab.
const { id: storeUid } = storeToRefs(usePersistUserStore())
const isOwner = computed(() => !!storeUid.value && userId.value === storeUid.value)

// The active top-level tab = the 3rd URL segment (/user/:id/<seg>/…). Drives the
// KunTab highlight; defaults to 动态 (the landing tab) for the bare /user/:id.
const activeMainTab = computed(() => {
  const m = route.path.match(/^\/user\/\d+\/([^/]+)/)
  return m ? m[1]! : 'activity'
})

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
  <!-- Single REAL root box (NOT `display: contents`): it is the page-transition
       root — a box-less root drops the enter animation and warns "does not have
       a single root node". Keep any comment INSIDE the root. The profile is now
       a full-width identity header + a horizontal tab strip + the active sub-tab
       (动态 is the landing tab); the document scrolls (no inner scroll pane). -->
  <div class="space-y-4">
    <template v-if="!isBanned">
      <template v-if="data">
        <UserProfileHeader :user="data" />

        <!-- 左下 = the tab rail, 右下 = the content area. Desktop: a sticky
             vertical rail beside the content; mobile: a horizontal scrollable
             strip stacked on top (the sm:hidden rail takes no grid track). -->
        <div class="grid grid-cols-1 items-start gap-4 sm:grid-cols-[auto_minmax(0,1fr)]">
          <!-- Mobile: a single-row nav strip inside KunScrollShadow, which
               scrolls horizontally and fades its edges (box-shadow) to signal
               there's more — a plain hidden scroll is easy to miss on touch.
               KunScrollShadow must own the scroll, so the row is a w-fit /
               whitespace-nowrap flex that overflows it (the forum's established
               pattern, see galgame/Tag). -->
          <div class="sm:hidden">
            <KunScrollShadow axis="horizontal" shadow-size="2rem">
              <div class="flex w-fit items-center gap-2 whitespace-nowrap py-1">
                <KunButton
                  v-for="tab in kunUserMainNav(data.id, isOwner)"
                  :key="tab.value"
                  :href="tab.href"
                  size="sm"
                  :variant="activeMainTab === tab.value ? 'flat' : 'light'"
                  :color="activeMainTab === tab.value ? 'primary' : 'default'"
                  class-name="shrink-0 gap-1.5"
                >
                  <KunIcon v-if="tab.icon" :name="tab.icon" />
                  {{ tab.textValue }}
                </KunButton>
              </div>
            </KunScrollShadow>
          </div>

          <!-- top-36 (144px), not flush at top-[7.5rem]: the collapsed mini
               header bar is fixed at top-20 and ends ~120px down, so this leaves
               a ~24px breathing gap between that bar and the sticky rail. -->
          <div class="hidden self-start sm:sticky sm:top-36 sm:block">
            <KunTab
              :items="kunUserMainNav(data.id, isOwner)"
              :model-value="activeMainTab"
              orientation="vertical"
              variant="underlined"
              color="primary"
              align="start"
            />
          </div>

          <!-- min-height keeps the grid (the sticky rail's containing block)
               at least a viewport tall, so the rail actually has room to stick
               even when a sub-tab's content is short — otherwise the grid would
               be only as tall as the rail and it would scroll away. -->
          <div class="min-w-0 sm:min-h-[calc(100dvh-9rem)]">
            <NuxtPage :user="data" />
          </div>
        </div>
      </template>

      <KunNull v-else description="未找到该用户" />
    </template>

    <KunNull v-else description="此用户已被封禁" />
  </div>
</template>
