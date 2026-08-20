<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

const props = defineProps<{
  user: UserInfo
}>()

const { id } = usePersistUserStore()
const isOwner = computed(() => id === props.user.id)

useKunDisableSeo('游玩时长')
</script>

<template>
  <UserPlaytime v-if="isOwner" />
  <KunNull v-else description="游玩时长只对本人可见" />
</template>
