<script setup lang="ts">
defineProps<{
  message: MessageSystemMessage
}>()
</script>

<template>
  <div
    class="space-y-3 rounded-lg p-2"
    :class="message.is_read ? 'message-read' : ''"
  >
    <div class="flex items-center break-all">
      <div class="ml-2 text-lg">
        <KunIcon
          class="text-secondary"
          v-if="!message.is_read"
          name="lucide:info"
        />
        <KunIcon
          class="text-default"
          v-if="message.is_read"
          name="lucide:check-check"
        />
      </div>
      <KunAvatar :disable-floating="true" :user="message.admin" />
      <span class="text-default-500 text-sm">
        <KunTime :time="message.created" />
      </span>
    </div>

    <div class="leading-8 break-all" v-html="message.content" />
  </div>
</template>
