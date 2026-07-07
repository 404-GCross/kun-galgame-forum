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

// 查看详情 overlay: only one open at a time; answerer records are fetched lazily
// on first open and cached per quiz.
const expandedId = ref<number | null>(null)
const records = ref<Record<number, QuizAnswererRecord[]>>({})
const loadingId = ref<number | null>(null)

const toggleDetail = async (quiz: GalgameQuizCard) => {
  if (expandedId.value === quiz.id) {
    expandedId.value = null
    return
  }
  expandedId.value = quiz.id
  if (!records.value[quiz.id]) {
    loadingId.value = quiz.id
    const data = await kunFetch<QuizAnswererRecord[]>(
      `/galgame-quiz/${quiz.id}/answers`
    )
    records.value[quiz.id] = data ?? []
    loadingId.value = null
  }
}
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
      <div class="relative flex h-full flex-col gap-3">
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
              {{
                quiz.galgame
                  ? getPreferredLanguageText(quiz.galgame.name)
                  : KUN_QUIZ_CATEGORY_MAP[quiz.category]
              }}
            </KunChip>
          </div>

          <div class="flex justify-end">
            <KunButton
              variant="light"
              size="sm"
              @click.stop.prevent="toggleDetail(quiz)"
            >
              <span class="flex items-center gap-1">
                <KunIcon name="lucide:list-tree" />查看详情
              </span>
            </KunButton>
          </div>
        </div>

        <!-- 查看详情 overlay — v-show reveals the otherwise-hidden details over
             the card. Uses shadow + a transparent→background top fade. That
             gradient is a SANCTIONED EXCEPTION to 铁律 #2 (no-gradient),
             explicitly authorized for this panel. -->
        <div
          v-show="expandedId === quiz.id"
          class="border-default-200 absolute inset-0 z-10 flex flex-col overflow-hidden rounded-xl border bg-[oklch(var(--content1))] shadow-xl"
          @click.stop.prevent
        >
          <div
            class="pointer-events-none absolute inset-x-0 top-0 z-10 h-6 bg-gradient-to-b from-[oklch(var(--content1))] to-transparent"
          />

          <div class="flex items-center justify-between px-3 pt-3 pb-1">
            <span class="text-sm font-medium">本题详情</span>
            <KunButton
              :is-icon-only="true"
              variant="light"
              size="sm"
              @click.stop.prevent="expandedId = null"
            >
              <KunIcon name="lucide:x" />
            </KunButton>
          </div>

          <div class="flex-1 space-y-3 overflow-y-auto px-3 pb-3 text-sm">
            <div>
              <p class="text-default-400 mb-1 text-xs">关联 Galgame</p>
              <KunLink
                v-if="quiz.galgame"
                :to="`/galgame/${quiz.galgame.id}`"
                @click.stop
              >
                {{ getPreferredLanguageText(quiz.galgame.name) }}
              </KunLink>
              <span v-else class="text-default-500">
                通用题（未关联具体作品）·
                {{ KUN_QUIZ_CATEGORY_MAP[quiz.category] }}
              </span>
            </div>

            <div>
              <p class="text-default-400 mb-1 text-xs">作答统计</p>
              <div class="flex flex-wrap items-center gap-3">
                <span class="text-success flex items-center gap-1">
                  <KunIcon name="lucide:check" />对 {{ quiz.correct_count }}
                </span>
                <span class="text-danger flex items-center gap-1">
                  <KunIcon name="lucide:x" />错
                  {{ quiz.answer_count - quiz.correct_count }}
                </span>
                <span class="text-default-500">共 {{ quiz.answer_count }} 人</span>
                <span v-if="quiz.quality_count > 0" class="text-default-500">
                  质量 {{ quiz.quality_average }}
                </span>
              </div>
            </div>

            <div>
              <p class="text-default-400 mb-1 text-xs">作答记录</p>
              <p v-if="loadingId === quiz.id" class="text-default-500">
                加载中…
              </p>
              <p
                v-else-if="!records[quiz.id]?.length"
                class="text-default-500"
              >
                还没有人作答
              </p>
              <div v-else class="space-y-1">
                <div
                  v-for="(rec, i) in records[quiz.id]"
                  :key="i"
                  class="flex items-center gap-2"
                >
                  <KunAvatar
                    :disable-floating="true"
                    :user="rec.user"
                    size="xs"
                    :is-navigation="false"
                  />
                  <span class="text-default-700 min-w-0 flex-1 truncate">
                    {{ rec.user.name }}
                  </span>
                  <KunIcon
                    v-if="rec.is_correct === true"
                    name="lucide:check"
                    class="text-success shrink-0"
                  />
                  <KunIcon
                    v-else-if="rec.is_correct === false"
                    name="lucide:x"
                    class="text-danger shrink-0"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </KunCard>
  </div>
</template>
