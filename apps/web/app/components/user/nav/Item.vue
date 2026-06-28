<script setup lang="ts">
import { KUN_USER_PAGE_NAV_MAP } from '~/constants/user'
import type { Nav } from '../utils/routeName'

const { id: storeUid } = storeToRefs(usePersistUserStore())

const props = defineProps<{
  userId: number
  nav: Nav[]
}>()

const route = useRoute()

const currentPageUid = computed(() => props.userId)
const activeRouteName = computed(() => {
  // third part
  const match = route.fullPath.match(/^\/user\/\d+\/([^/]+)/)
  return match ? match[1] : ''
})

const iconMap: Record<string, string> = {
  profile: 'lucide:user-round',
  setting: 'lucide:settings',
  topic: 'lucide:square-gantt-chart',
  galgame: 'lucide:gamepad-2',
  resource: 'lucide:package',
  rating: 'lucide:star',
  reply: 'carbon:reply',
  comment: 'uil:comment-dots'
}

// Owner = the logged-in viewer is looking at their OWN profile (id match,
// never a role). Gates the owner-only nav entries (e.g. 设置).
const isOwner = computed(
  () => !!storeUid.value && props.userId === storeUid.value
)

const handleCollapsed = async (item: Nav) => {
  if (item.collapsed !== undefined) {
    item.collapsed = !item.collapsed
  } else {
    await navigateTo(
      `/user/${currentPageUid.value}/${item.redirect ? item.redirect : item.router}`
    )
  }
}
</script>

<template>
  <div
    class="scrollbar-hide h-full w-9 shrink-0 space-y-1 overflow-y-auto sm:w-40"
  >
    <div
      v-for="(kun, index) in nav"
      :key="index"
      v-show="!kun.ownerOnly || isOwner"
    >
      <KunButton
        :full-width="true"
        :variant="activeRouteName === kun.router ? 'flat' : 'light'"
        size="lg"
        class-name="gap-2 justify-start text-foreground sm:px-5 sm:py-2.5 p-2"
        @click="handleCollapsed(kun)"
      >
        <KunIcon
          class="shrink-0 text-xl text-inherit sm:text-base"
          v-if="iconMap[kun.name]"
          :name="iconMap[kun.name]"
        />
        <span class="hidden text-inherit sm:block">
          {{ KUN_USER_PAGE_NAV_MAP[kun.name] }}
        </span>
      </KunButton>
    </div>
  </div>
</template>
