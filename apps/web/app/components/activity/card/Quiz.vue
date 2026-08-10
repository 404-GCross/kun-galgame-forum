<script setup lang="ts">
import { kunQuizDifficultyLabel } from '~/constants/galgame-quiz'

const props = defineProps<{ activity: ActivityItem }>()

const data = computed(() => props.activity.data as QuizActivityData | undefined)
const quizId = computed(
  () => Number(props.activity.link.split('/').pop()) || 0
)

const summary = computed(() => {
  const d = data.value
  if (!d) return '出了一道题目。'
  return `出了一道题目，${kunQuizDifficultyLabel(d.difficulty)}·难度${d.difficulty}，已经有 ${d.answer_count} 人作答。`
})

const descriptionText = computed(() => {
  const d = data.value?.description
  return d ? markdownToText(d) : ''
})

const { isFavorited, setFavorited, ensureLoaded } = useMyQuizInteractions()
onMounted(ensureLoaded)
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-2">
      <p class="text-default-600 text-sm">{{ summary }}</p>

      <KunLink
        underline="none"
        color="default"
        :to="activity.link"
        class-name="group block"
      >
        <p
          class="group-hover:text-primary line-clamp-3 text-base break-words transition-colors"
        >
          {{ activity.content }}
        </p>
      </KunLink>

      <p
        v-if="descriptionText"
        class="text-default-500 text-sm break-words whitespace-pre-line"
      >
        {{ descriptionText }}
      </p>

      <div class="flex items-center gap-2">
        <FavoriteToggle
          :favorited="isFavorited(quizId)"
          :count="data?.favorite_count ?? 0"
          :endpoint="`/galgame-quiz/${quizId}/favorite`"
          size="sm"
          @changed="(v: boolean) => setFavorited(quizId, v)"
        />
        <KunLink
          underline="none"
          color="default"
          :to="activity.link"
          class-name="text-default-500 hover:text-primary ml-auto flex items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
