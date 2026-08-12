<script setup lang="ts">
const props = defineProps<{
  id: number
}>()

const res = await kunFetch<{ name: string; avatar: string }>(
  `/user/${props.id}`,
  {
    method: 'GET',
    query: { user_id: props.id }
  }
)

const user = {
  id: props.id,
  name: res ? res.name : '',
  avatar: res ? res.avatar : ''
}
</script>

<template>
  <header class="flex items-center gap-2">
    <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
      <KunIcon name="lucide:chevron-left" />
    </KunButton>

    <KunAvatar :disable-floating="true" :user="user" />

    <h2 class="relative flex items-center gap-2">
      <span>{{ user.name }}</span>
    </h2>
  </header>
</template>
