<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_CONST,
  KUN_QUIZ_ENABLED_TYPE_CONST,
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
  KUN_QUIZ_TYPE_DESCRIPTION_MAP,
  KUN_QUIZ_CATEGORY_CONST,
  KUN_QUIZ_CATEGORY_MAP,
  KUN_QUIZ_SPOILER_CONST,
  KUN_QUIZ_SPOILER_MAP,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'
import { createGalgameQuizSchema } from '~/validations/galgame-quiz'

const props = defineProps<{
  modelValue: boolean
  // Optional: bind the quiz to a specific game (passed from a galgame page).
  galgameId?: number
  // When present, the modal is in EDIT mode (pre-fill + PUT).
  editData?: QuizEditData | null
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  onPublished: [quiz: GalgameQuizCard]
  onUpdated: []
}>()

const category = ref<QuizCategory>('trivia')
const type = ref<QuizType>('single')
const difficulty = ref(3)
const spoilerLevel = ref<QuizSpoilerLevel>('none')
const question = ref('')
const explanation = ref('')
// The galgame chosen via the picker (only when not pre-bound by the galgameId prop).
const pickedGalgameId = ref<number | null>(null)
// Progressive disclosure: the optional explanation stays hidden until asked for.
// (关联 Galgame is important, so it's shown up-front, not collapsed.)
const showExplanation = ref(false)
const isSubmitting = ref(false)

const editorRef = ref<{
  getContent: () => Record<string, unknown>
  validate: () => string | null
  reset: () => void
  load: (content: Record<string, unknown>) => void
} | null>(null)

const initialSelected = ref<{ id: number; name: string } | null>(null)
const isEditing = computed(() => !!props.editData)

// Edit mode: pre-fill the whole form + editor when the modal opens.
watch(
  () => props.modelValue,
  async (open) => {
    if (!open || !props.editData) return
    const d = props.editData
    category.value = d.category
    type.value = d.type
    difficulty.value = d.difficulty
    spoilerLevel.value = d.spoiler_level
    question.value = d.question
    explanation.value = d.explanation
    showExplanation.value = !!d.explanation
    pickedGalgameId.value = d.galgame_id
    initialSelected.value = d.galgame
      ? {
          id: d.galgame.id,
          name: getPreferredLanguageText(d.galgame.name) || `#${d.galgame.id}`
        }
      : null
    await nextTick()
    if (!editorRef.value) await nextTick()
    editorRef.value?.load(d.content as unknown as Record<string, unknown>)
  }
)

// 题型 as a pill KunCheckBoxGroup (like the topic 分区 selector). Type is
// single-select, so the group's array v-model is adapted to a single value:
// picking a new pill keeps only the latest, and the selected pill can't be
// unchecked (a type is always required). fill/essay are shown disabled.
const typeGroupOptions = KUN_QUIZ_TYPE_CONST.map((t) => {
  const enabled = (KUN_QUIZ_ENABLED_TYPE_CONST as readonly string[]).includes(t)
  return {
    value: t,
    label: KUN_QUIZ_TYPE_MAP[t] ?? t,
    icon: KUN_QUIZ_TYPE_ICON_MAP[t] ?? 'lucide:circle',
    disabled: !enabled
  }
})
const typeSelection = computed<QuizType[]>({
  get: () => [type.value],
  set: (arr) => {
    const last = arr[arr.length - 1]
    if (last) type.value = last
  }
})

const categoryOptions = KUN_QUIZ_CATEGORY_CONST.map((c) => ({
  value: c,
  label: KUN_QUIZ_CATEGORY_MAP[c] ?? c
}))
// 分类 as a single-select pill KunCheckBoxGroup (same array adapter as 题型).
const categorySelection = computed<QuizCategory[]>({
  get: () => [category.value],
  set: (arr) => {
    const last = arr[arr.length - 1]
    if (last) category.value = last
  }
})

const spoilerOptions = KUN_QUIZ_SPOILER_CONST.map((s) => ({
  value: s,
  label: KUN_QUIZ_SPOILER_MAP[s] ?? s
}))

const close = () => emits('update:modelValue', false)

const resetForm = () => {
  category.value = 'trivia'
  type.value = 'single'
  difficulty.value = 3
  spoilerLevel.value = 'none'
  question.value = ''
  explanation.value = ''
  pickedGalgameId.value = null
  initialSelected.value = null
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
    spoiler_level: spoilerLevel.value,
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

  // Edit mode → PUT; otherwise create → POST.
  if (isEditing.value && props.editData) {
    body.quiz_id = props.editData.id
    isSubmitting.value = true
    const ok = await kunFetch(`/galgame-quiz/${props.editData.id}`, {
      method: 'PUT',
      body
    })
    isSubmitting.value = false
    if (ok) {
      useMessage('已保存修改', 'success')
      emits('onUpdated')
      close()
    }
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
        <KunHeader :name="isEditing ? '编辑题目' : '出题'" scale="h3" />
        <p
          v-if="!isEditing"
          class="text-default-400 flex items-center gap-1 text-xs"
        >
          <KunIcon name="lucide:lollipop" />出题即得 2 萌萌点 · 题目被删除时扣除
        </p>
      </div>

      <!-- 关联 Galgame — important, so it sits up top and stays visible -->
      <KunInfo
        v-if="galgameId"
        color="primary"
        icon="lucide:link"
        title="已关联当前 Galgame"
        description="本题将关联到你当前所在的 Galgame"
      />
      <GalgameQuizGalgamePicker
        v-else
        v-model="pickedGalgameId"
        :initial-selected="initialSelected"
      />

      <!-- 题型 (pill KunCheckBoxGroup, single-select) -->
      <div class="space-y-2">
        <div class="flex items-center gap-2">
          <label class="text-sm font-medium">题型</label>
          <span class="text-default-400 text-xs">填空、问答即将实装</span>
        </div>
        <KunCheckBoxGroup
          v-model="typeSelection"
          :options="typeGroupOptions"
          variant="pill"
          color="primary"
          size="sm"
          orientation="horizontal"
        />
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

      <!-- 4) 属性: 分类 (pill) + 难度 / 剧透等级 -->
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium">分类</label>
          <KunCheckBoxGroup
            v-model="categorySelection"
            :options="categoryOptions"
            variant="pill"
            color="primary"
            size="sm"
            orientation="horizontal"
          />
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
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

          <KunSelect
            v-model="spoilerLevel"
            :options="spoilerOptions"
            label="剧透等级"
          />
        </div>
      </div>

      <!-- 解析 (progressive) -->
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
        <KunButton :loading="isSubmitting" @click="submit">
          {{ isEditing ? '保存修改' : '发布题目' }}
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
