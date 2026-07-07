<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
  KUN_QUIZ_TYPE_COLOR_MAP,
  KUN_QUIZ_CATEGORY_MAP,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'
import { answerGalgameQuizSchema } from '~/validations/galgame-quiz'

const props = defineProps<{ quiz: GalgameQuizPlay }>()

// Local mutable copy so answering / rating updates the view without a refetch.
const state = ref<GalgameQuizPlay>({ ...props.quiz })
const { canModerate } = useRole()

const answerRef = ref<{
  getSubmitted: () => Record<string, unknown>
  validate: () => string | null
} | null>(null)
const isSubmitting = ref(false)

const submitAnswer = async () => {
  if (!requireLogin()) return
  const err = answerRef.value?.validate()
  if (err) {
    useMessage(err, 'warn')
    return
  }
  const submitted = answerRef.value?.getSubmitted() ?? {}
  const body = { quiz_id: state.value.id, submitted }
  const valid = useKunSchemaValidator(answerGalgameQuizSchema, body)
  if (!valid) return

  isSubmitting.value = true
  const res = await kunFetch<QuizAnswerResult>(
    `/galgame-quiz/${state.value.id}/answer`,
    { method: 'POST', body }
  )
  isSubmitting.value = false
  if (res) {
    state.value.my_answer = res
    state.value.answer_count += 1
    if (res.is_correct) state.value.correct_count += 1
    if (res.reward_delta > 0) {
      useMessage(`回答正确, +${res.reward_delta} 萌萌点`, 'success')
    }
  }
}

const onRated = (r: QuizQualityResult) => {
  state.value.quality_average = r.quality_average
  state.value.quality_count = r.quality_count
  if (state.value.my_answer) {
    state.value.my_answer.quality_rating = r.quality_rating
  }
}

const isDeleting = ref(false)
const canDelete = computed(() => state.value.is_author || canModerate.value)
const remove = async () => {
  const ok = await useComponentMessageStore().alert(
    '确认删除',
    '删除后本题及所有作答记录将被移除, 出题获得的萌萌点会被扣除'
  )
  if (!ok) return
  isDeleting.value = true
  const res = await kunFetch(
    `/galgame-quiz/${state.value.id}?quiz_id=${state.value.id}`,
    { method: 'DELETE' }
  )
  isDeleting.value = false
  if (res) {
    useMessage('已删除', 'success')
    navigateTo('/galgame-quiz')
  }
}

const correctRate = computed(() =>
  state.value.answer_count > 0
    ? Math.round((state.value.correct_count / state.value.answer_count) * 100)
    : null
)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2">
      <KunButton variant="light" size="sm" href="/galgame-quiz">
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:arrow-left" />返回题库
        </span>
      </KunButton>

      <KunButton
        v-if="canDelete"
        variant="light"
        color="danger"
        size="sm"
        :loading="isDeleting"
        @click="remove"
      >
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:trash-2" />删除
        </span>
      </KunButton>
    </div>

    <KunCard :is-transparent="false">
      <div class="space-y-4">
        <div class="flex flex-wrap items-center gap-2">
          <KunChip :color="KUN_QUIZ_TYPE_COLOR_MAP[state.type]" variant="flat">
            <span class="flex items-center gap-1">
              <KunIcon :name="KUN_QUIZ_TYPE_ICON_MAP[state.type]" />
              {{ KUN_QUIZ_TYPE_MAP[state.type] }}
            </span>
          </KunChip>
          <KunChip
            :color="kunQuizDifficultyColor(state.difficulty)"
            variant="solid"
          >
            {{ kunQuizDifficultyLabel(state.difficulty) }} · {{ state.difficulty }}
          </KunChip>
          <KunChip variant="light">
            {{ KUN_QUIZ_CATEGORY_MAP[state.category] }}
          </KunChip>
          <KunLink
            v-if="state.galgame"
            :to="`/galgame/${state.galgame.id}`"
            class="text-sm"
          >
            {{ getPreferredLanguageText(state.galgame.name) }}
          </KunLink>
        </div>

        <h1 class="text-xl font-bold break-words whitespace-pre-wrap">
          {{ state.question }}
        </h1>

        <div
          class="text-default-500 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm"
        >
          <span class="text-default-700 flex items-center gap-1">
            <KunAvatar
              :disable-floating="true"
              :user="state.user"
              size="xs"
              :is-navigation="false"
            />
            {{ state.user.name }}
          </span>
          <KunTime :time="state.created" />
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:users" />{{ state.answer_count }} 人作答
          </span>
          <span v-if="correctRate !== null" class="flex items-center gap-1">
            <KunIcon name="lucide:target" />正确率 {{ correctRate }}%
          </span>
        </div>

        <KunDivider />

        <!-- answer flow -->
        <div v-if="!state.my_answer" class="space-y-4">
          <GalgameQuizPlayAnswerInput
            ref="answerRef"
            :type="state.type"
            :content="state.content"
          />
          <div class="flex justify-end">
            <KunButton :loading="isSubmitting" @click="submitAnswer">
              {{ state.type === 'essay' ? '提交并查看参考答案' : '提交答案' }}
            </KunButton>
          </div>
        </div>

        <!-- result / reveal -->
        <GalgameQuizPlayResult
          v-else
          :quiz="state"
          :result="state.my_answer"
          @rated="onRated"
        />
      </div>
    </KunCard>
  </div>
</template>
