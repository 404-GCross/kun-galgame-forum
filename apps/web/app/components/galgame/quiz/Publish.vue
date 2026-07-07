<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_CONST,
  KUN_QUIZ_ENABLED_TYPE_CONST,
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
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
// The galgame chosen via the picker (only when not pre-bound by the galgameId prop).
const pickedGalgameId = ref<number | null>(null)
// Progressive disclosure: the optional link + explanation stay hidden until asked for.
const showLink = ref(false)
const showExplanation = ref(false)
const isSubmitting = ref(false)

const editorRef = ref<{
  getContent: () => Record<string, unknown>
  validate: () => string | null
  reset: () => void
} | null>(null)

// 题型 as a segmented tile row — all five shown; not-yet-implemented ones
// (fill/essay) rendered disabled with a hint rather than removed.
const typeTiles = KUN_QUIZ_TYPE_CONST.map((t) => ({
  value: t,
  label: KUN_QUIZ_TYPE_MAP[t] ?? t,
  icon: KUN_QUIZ_TYPE_ICON_MAP[t] ?? 'lucide:circle',
  enabled: (KUN_QUIZ_ENABLED_TYPE_CONST as readonly string[]).includes(t)
}))
const selectType = (tile: { value: QuizType; enabled: boolean }) => {
  if (tile.enabled) type.value = tile.value
}

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
  pickedGalgameId.value = null
  showLink.value = false
  showExplanation.value = false
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
  // Pre-bound (opened from a galgame page) wins; otherwise use the picker.
  const linkedGalgameId = props.galgameId ?? pickedGalgameId.value
  if (linkedGalgameId) {
    body.galgame_id = linkedGalgameId
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
    inner-class-name="max-w-[720px] w-[90vw]"
    :is-dismissable="false"
    @update:model-value="(v) => emits('update:modelValue', v)"
  >
    <div class="space-y-5">
      <div class="space-y-1">
        <KunHeader name="出题" scale="h3" />
        <p class="text-default-400 flex items-center gap-1 text-xs">
          <KunIcon name="lucide:lollipop" />出题即得 2 萌萌点 · 题目被删除时扣除
        </p>
      </div>

      <!-- 1) 题型 (segmented tiles) -->
      <div class="space-y-2">
        <div class="grid grid-cols-3 gap-2 sm:grid-cols-5">
          <button
            v-for="tile in typeTiles"
            :key="tile.value"
            type="button"
            :disabled="!tile.enabled"
            class="flex flex-col items-center gap-1 rounded-xl border-2 px-2 py-3 text-sm transition-colors"
            :class="[
              type === tile.value
                ? 'border-primary bg-primary/10 text-primary'
                : 'border-default-200 text-default-600 hover:border-default-300',
              !tile.enabled && 'cursor-not-allowed opacity-40 hover:border-default-200'
            ]"
            @click="selectType(tile)"
          >
            <KunIcon :name="tile.icon" class="size-5" />
            <span>{{ tile.label }}</span>
            <span v-if="!tile.enabled" class="text-[10px] leading-none">
              即将实装
            </span>
          </button>
        </div>
        <p class="text-default-500 text-sm">
          {{ KUN_QUIZ_TYPE_DESCRIPTION_MAP[type] }}
        </p>
      </div>

      <!-- 2) 题干 -->
      <KunTextarea
        v-model="question"
        label="题目"
        :rows="2"
        placeholder="例如: 《永不枯萎的世界与终结之花》中莲什么时候来过月经"
        :maxlength="2000"
        :show-char-count="true"
        auto-grow
      />

      <!-- 3) 选项 / 答案 (correctness inline) -->
      <GalgameQuizContentEditor ref="editorRef" :type="type" />

      <KunDivider />

      <!-- 4) 属性: 分类 + 难度 -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <KunSelect v-model="category" :options="categoryOptions" label="分类" />
        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <label class="text-sm">难度</label>
            <KunChip
              :color="kunQuizDifficultyColor(difficulty)"
              variant="flat"
              size="sm"
            >
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
      </div>

      <!-- 5) 关联 Galgame (progressive) -->
      <KunInfo
        v-if="galgameId"
        color="primary"
        icon="lucide:link"
        title="已关联当前 Galgame"
        description="本题将关联到你当前所在的 Galgame"
      />
      <GalgameQuizGalgamePicker
        v-else-if="showLink || pickedGalgameId !== null"
        v-model="pickedGalgameId"
      />
      <KunButton v-else variant="light" size="sm" @click="showLink = true">
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:link" />关联 Galgame（可选）
        </span>
      </KunButton>

      <!-- 6) 解析 (progressive) -->
      <KunTextarea
        v-if="showExplanation || explanation"
        v-model="explanation"
        label="解析（可选, 作答后展示）"
        :rows="2"
        placeholder="可以补充答案的解析、出处或冷知识"
        :maxlength="2000"
        :show-char-count="true"
        auto-grow
      />
      <KunButton
        v-else
        variant="light"
        size="sm"
        @click="showExplanation = true"
      >
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:plus" />添加解析（可选）
        </span>
      </KunButton>

      <div class="flex items-center justify-end gap-2 pt-1">
        <KunButton variant="light" color="default" @click="close">取消</KunButton>
        <KunButton :loading="isSubmitting" @click="submit">发布题目</KunButton>
      </div>
    </div>
  </KunModal>
</template>
