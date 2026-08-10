<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

const route = useRoute()

const userId = parseInt((route.params as { id: string }).id)

useKunDisableSeo('私信')
</script>

<template>
  <div class="flex h-full min-w-0 flex-1 flex-col pl-3">
    <ClientOnly>
      <MessagePmHeader :id="userId" />
    </ClientOnly>

    <MessagePmContainer :user-id="userId" />
  </div>
</template>
