<script setup lang="ts">
const route = useRoute()
const gid = computed(() => parseInt((route.params as { gid: string }).gid))

const { data } = await useKunFetch<GalgameLink[]>(
  `/galgame/${gid.value}/link/all`,
  {
    lazy: true,
    method: 'GET',
    query: { galgame_id: gid.value },
    watch: false
  }
)
</script>

<template>
  <div v-if="data?.length" class="flex flex-wrap gap-x-3 gap-y-1">
    <KunLink
      v-for="(link, index) in data"
      :key="index"
      :to="link.link"
      target="_blank"
      rel="noopener noreferrer"
      size="sm"
      color="default"
      class-name="text-default-500 hover:text-default-700"
    >
      {{ link.name }}
    </KunLink>
  </div>
</template>
