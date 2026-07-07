<script setup lang="ts">
const route = useRoute()
const { data } = await useKunFetch<GalgameQuizPlay>(
  `/galgame-quiz/${route.params.id}`
)

useKunSeoMeta({
  title: data.value ? `Galgame 答题 · ${data.value.question.slice(0, 30)}` : 'Galgame 答题',
  description: data.value?.question ?? '在 Galgame 题库中作答这道题目。'
})
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <GalgameQuizPlay v-if="data" :quiz="data" />
    <KunNull v-else description="题目不存在或已被删除" />
  </div>
</template>
