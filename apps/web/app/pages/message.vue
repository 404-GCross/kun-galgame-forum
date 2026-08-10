<script setup lang="ts">
definePageMeta({
  middleware: 'auth',
  pageTransition: false
})

const { messageStatus } = storeToRefs(useTempSettingStore())

useHead({ title: `消息 - ${kungal.titleShort}` })

onMounted(() => (messageStatus.value = 'online'))
</script>

<template>
  <div class="flex h-[calc(100dvh-120px)] flex-row sm:gap-4">
    <MessageAsideContainer />
    <NuxtPage :transition="false" />
  </div>
</template>
