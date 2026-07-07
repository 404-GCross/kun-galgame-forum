<script setup lang="ts">
import {
  KUN_QUIZ_ENABLED_TYPE_CONST,
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_DESCRIPTION_MAP,
  KUN_QUIZ_CATEGORY_CONST,
  KUN_QUIZ_CATEGORY_MAP,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'
import { createGalgameQuizSchema } from '~/validations/galgame-quiz'

const props = defineProps<{
  modelValue: boolean
  // Optional: bind the quiz to a specific game (passed from a galgame page).
  galgameId?: number
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  onPublished: [quiz: GalgameQuizCard]
}>()

const category = ref<QuizCategory>('trivia')
const type = ref<QuizType>('single')
const difficulty = ref(3)
const question = ref('')
const explanation = ref('')
const isSubmitting = ref(false)

const editorRef = ref<{
  getContent: () => Record<string, unknown>
  validate: () => string | null
  reset: () => void
} | null>(null)

const typeOptions = KUN_QUIZ_ENABLED_TYPE_CONST.map((t) => ({
  value: t,
  label: KUN_QUIZ_TYPE_MAP[t] ?? t
}))
const categoryOptions = KUN_QUIZ_CATEGORY_CONST.map((c) => ({
  value: c,
  label: KUN_QUIZ_CATEGORY_MAP[c] ?? c
}))

const close = () => emits('update:modelValue', false)

const resetForm = () => {
  category.value = 'trivia'
  type.value = 'single'
  difficulty.value = 3
  question.value = ''
  explanation.value = ''
  editorRef.value?.reset()
}

const submit = async () => {
  const contentError = editorRef.value?.validate()
  if (contentError) {
    useMessage(contentError, 'warn')
    return
  }
  const content = editorRef.value?.getContent() ?? {}

  const body: Record<string, unknown> = {
    category: category.value,
    type: type.value,
    difficulty: difficulty.value,
    question: question.value,
    content,
    explanation: explanation.value
  }
  if (props.galgameId) {
    body.galgame_id = props.galgameId
  }

  const valid = useKunSchemaValidator(createGalgameQuizSchema, body)
  if (!valid) {
    return
  }

  isSubmitting.value = true
  const res = await kunFetch<GalgameQuizCard>('/galgame-quiz', {
    method: 'POST',
    body
  })
  isSubmitting.value = false
  if (res) {
    useMessage('出题成功', 'success')
    emits('onPublished', res)
    resetForm()
    close()
  }
}
</script>

<template>
  <KunModal
    :model-value="modelValue"
    inner-class-name="max-w-[780px] w-[90vw]"
    :is-dismissable="false"
    @update:model-value="(v) => emits('update:modelValue', v)"
  >
    <div class="space-y-4">
      <KunHeader
        name="出题"
        description="出一道优质的 Galgame 题目, 合格出题将获得萌萌点; 题目被删除时萌萌点会被扣除"
        scale="h3"
      />

      <KunInfo
        title="萌萌点奖励"
        color="secondary"
        icon="lucide:lollipop"
        description="出题即获得 2 萌萌点, 题目被删除时会扣除对应萌萌点"
      />

      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <KunSelect v-model="category" :options="categoryOptions" label="分类" />
        <KunSelect v-model="type" :options="typeOptions" label="题型" />
      </div>
      <p class="text-default-500 text-sm">
        {{ KUN_QUIZ_TYPE_DESCRIPTION_MAP[type] }}
      </p>

      <div class="space-y-1">
        <div class="flex items-center justify-between">
          <label class="text-sm font-medium">难度 (1-10)</label>
          <KunChip :color="kunQuizDifficultyColor(difficulty)" variant="flat">
            {{ kunQuizDifficultyLabel(difficulty) }} · {{ difficulty }}
          </KunChip>
        </div>
        <KunSlider
          v-model="difficulty"
          :min="1"
          :max="10"
          :step="1"
          :color="kunQuizDifficultyColor(difficulty)"
        />
      </div>

      <KunTextarea
        v-model="question"
        label="题干"
        :rows="3"
        placeholder="例如: 《永不枯萎的世界与终结之花》中莲什么时候来过月经"
        :maxlength="2000"
        :show-char-count="true"
        auto-grow
      />

      <GalgameQuizContentEditor ref="editorRef" :type="type" />

      <KunTextarea
        v-model="explanation"
        label="解析 (可选, 作答后展示)"
        :rows="2"
        placeholder="可以补充答案的解析、出处或冷知识"
        :maxlength="2000"
        :show-char-count="true"
        auto-grow
      />

      <div class="flex items-center justify-end gap-2">
        <KunButton variant="light" color="default" @click="close">取消</KunButton>
        <KunButton :loading="isSubmitting" @click="submit">发布题目</KunButton>
      </div>
    </div>
  </KunModal>
</template>
