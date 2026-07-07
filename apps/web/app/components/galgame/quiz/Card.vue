<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
  KUN_QUIZ_TYPE_COLOR_MAP,
  KUN_QUIZ_CATEGORY_MAP,
  KUN_QUIZ_SPOILER_MAP,
  KUN_QUIZ_SPOILER_COLOR_MAP,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'

withDefaults(
  defineProps<{
    quizzes: GalgameQuizCard[]
    // Transparent on the floating list background; pass false on solid surfaces.
    isTransparent?: boolean
  }>(),
  { isTransparent: true }
)

const correctRate = (q: GalgameQuizCard) =>
  q.answer_count > 0
    ? Math.round((q.correct_count / q.answer_count) * 100)
    : null
</script>

<template>
  <div
    class="grid grid-cols-1 gap-2 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
  >
    <KunCard
      v-for="quiz in quizzes"
      :key="quiz.id"
      :is-transparent="isTransparent"
      :is-hoverable="true"
      :href="`/galgame-quiz/${quiz.id}`"
    >
      <div class="flex h-full flex-col gap-3">
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-1">
            <KunChip :color="KUN_QUIZ_TYPE_COLOR_MAP[quiz.type]" variant="flat">
              <span class="flex items-center gap-1">
                <KunIcon :name="KUN_QUIZ_TYPE_ICON_MAP[quiz.type]" />
                {{ KUN_QUIZ_TYPE_MAP[quiz.type] }}
              </span>
            </KunChip>
            <KunChip
              v-if="quiz.spoiler_level !== 'none'"
              :color="KUN_QUIZ_SPOILER_COLOR_MAP[quiz.spoiler_level]"
              variant="flat"
              size="sm"
            >
              {{ KUN_QUIZ_SPOILER_MAP[quiz.spoiler_level] }}
            </KunChip>
          </div>
          <KunChip
            :color="kunQuizDifficultyColor(quiz.difficulty)"
            variant="solid"
          >
            {{ kunQuizDifficultyLabel(quiz.difficulty) }} {{ quiz.difficulty }}
          </KunChip>
        </div>

        <h2
          class="hover:text-primary line-clamp-3 min-h-[4.5rem] font-medium break-words whitespace-pre-wrap transition-colors"
        >
          {{ quiz.question }}
        </h2>

        <div class="mt-auto flex flex-col gap-2">
          <div class="text-default-700 flex items-center gap-1 text-sm">
            <KunAvatar
              :disable-floating="true"
              :user="quiz.user"
              size="xs"
              :is-navigation="false"
            />
            {{ quiz.user.name }} · <KunTime :time="quiz.created" />
          </div>

          <div class="flex items-center justify-between text-sm">
            <div class="text-default-500 flex gap-3">
              <span class="flex items-center gap-1">
                <KunIcon name="lucide:users" />{{ quiz.answer_count }}
              </span>
              <span
                v-if="correctRate(quiz) !== null"
                class="flex items-center gap-1"
              >
                <KunIcon name="lucide:target" />{{ correctRate(quiz) }}%
              </span>
              <span
                v-if="quiz.quality_count > 0"
                class="flex items-center gap-1"
              >
                <KunIcon name="lucide:star" />{{ quiz.quality_average }}
              </span>
            </div>
            <KunChip size="sm" variant="light">
              {{ KUN_QUIZ_CATEGORY_MAP[quiz.category] }}
            </KunChip>
          </div>
        </div>
      </div>
    </KunCard>
  </div>
</template>
