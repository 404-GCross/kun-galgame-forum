<script setup lang="ts">
// Per-type authoring of the quiz `content` payload. The parent (Publish) holds
// a ref to this component and calls getContent() / validate() on submit — the
// server validates the same shape again, so this is UX-only.
//
// Correctness is marked INLINE on each option row (one action per row) rather
// than in a separate "correct answer" picker — the pattern used by Google
// Forms / Typeform / Kahoot.
const props = defineProps<{ type: QuizType }>()

// single / multiple
const options = ref<string[]>(['', ''])
const singleAnswer = ref(0)
const multiAnswers = ref<number[]>([])
// judge
const judgeAnswer = ref<'true' | 'false'>('true')
// fill: one string[] of accepted answers per blank
const blanks = ref<string[][]>([[]])
// essay
const essayReference = ref('')

const reset = () => {
  options.value = ['', '']
  singleAnswer.value = 0
  multiAnswers.value = []
  judgeAnswer.value = 'true'
  blanks.value = [[]]
  essayReference.value = ''
}
// Switching type starts a clean payload.
watch(() => props.type, reset)

const addOption = () => options.value.push('')
const removeOption = (i: number) => {
  if (options.value.length <= 2) return
  options.value.splice(i, 1)
  multiAnswers.value = multiAnswers.value
    .filter((a) => a !== i)
    .map((a) => (a > i ? a - 1 : a))
  if (singleAnswer.value >= options.value.length) singleAnswer.value = 0
}

// Inline correct-answer marking, driven by a per-row KunCheckBox
// (type="single" renders a radio, "multiple" a checkbox).
const isCorrect = (i: number) =>
  props.type === 'single'
    ? singleAnswer.value === i
    : multiAnswers.value.includes(i)
const setCorrect = (i: number, checked: boolean) => {
  if (props.type === 'single') {
    // radio semantics — this option becomes the sole answer
    singleAnswer.value = i
  } else if (checked) {
    if (!multiAnswers.value.includes(i)) multiAnswers.value.push(i)
  } else {
    multiAnswers.value = multiAnswers.value.filter((x) => x !== i)
  }
}

const addBlank = () => blanks.value.push([])
const removeBlank = (i: number) => {
  if (blanks.value.length <= 1) return
  blanks.value.splice(i, 1)
}

const getContent = (): Record<string, unknown> => {
  switch (props.type) {
    case 'single':
      return { options: options.value.map((o) => o.trim()), answer: singleAnswer.value }
    case 'multiple':
      return {
        options: options.value.map((o) => o.trim()),
        answers: [...multiAnswers.value].sort((a, b) => a - b)
      }
    case 'judge':
      return { answer: judgeAnswer.value === 'true' }
    case 'fill':
      return {
        blanks: blanks.value.map((accepted) => ({
          accepted: accepted.map((a) => a.trim()).filter(Boolean)
        }))
      }
    case 'essay':
      return { reference: essayReference.value.trim() }
    default:
      return {}
  }
}

// Returns an error message, or null when the payload is valid.
const validate = (): string | null => {
  switch (props.type) {
    case 'single':
    case 'multiple': {
      const opts = options.value.map((o) => o.trim())
      if (opts.length < 2) return '至少需要 2 个选项'
      if (opts.some((o) => !o)) return '选项内容不能为空'
      if (props.type === 'single' && singleAnswer.value >= opts.length)
        return '请标记正确答案'
      if (props.type === 'multiple' && multiAnswers.value.length === 0)
        return '请至少标记一个正确答案'
      return null
    }
    case 'judge':
      return null
    case 'fill': {
      if (blanks.value.length === 0) return '至少需要 1 个空'
      const bad = blanks.value.some(
        (accepted) => accepted.map((a) => a.trim()).filter(Boolean).length === 0
      )
      if (bad) return '每个空至少填写 1 个可接受答案'
      return null
    }
    case 'essay':
      return essayReference.value.trim() ? null : '请填写参考答案'
    default:
      return '未知题型'
  }
}

defineExpose({ getContent, validate, reset })
</script>

<template>
  <div class="space-y-3">
    <!-- single / multiple — correctness marked inline on each row -->
    <template v-if="type === 'single' || type === 'multiple'">
      <div class="flex items-center justify-between">
        <label class="text-sm font-medium">选项</label>
        <span class="text-default-400 text-xs">
          在左侧勾选正确答案（{{ type === 'single' ? '单选题唯一' : '多选题可多个' }}）
        </span>
      </div>

      <div class="space-y-2">
        <div
          v-for="(_, i) in options"
          :key="i"
          class="flex items-center gap-2"
        >
          <KunCheckBox
            :type="type === 'single' ? 'single' : 'multiple'"
            :model-value="isCorrect(i)"
            size="lg"
            color="success"
            @update:model-value="(v) => setCorrect(i, v)"
          />

          <KunInput
            v-model="options[i]"
            :placeholder="`选项 ${i + 1}`"
            class-name="grow"
          />

          <KunButton
            :is-icon-only="true"
            variant="light"
            color="danger"
            size="sm"
            :disabled="options.length <= 2"
            @click="removeOption(i)"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </div>
      </div>

      <KunButton variant="light" size="sm" @click="addOption">
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:plus" />添加选项
        </span>
      </KunButton>
    </template>

    <!-- judge — two big choice cards -->
    <template v-else-if="type === 'judge'">
      <label class="text-sm font-medium">正确答案</label>
      <div class="grid grid-cols-2 gap-3">
        <button
          type="button"
          class="flex items-center justify-center gap-2 rounded-xl border-2 py-5 text-lg font-medium transition-colors"
          :class="
            judgeAnswer === 'true'
              ? 'border-success bg-success/10 text-success'
              : 'border-default-200 text-default-500 hover:border-default-300'
          "
          @click="judgeAnswer = 'true'"
        >
          <KunIcon name="lucide:circle-check" />正确
        </button>
        <button
          type="button"
          class="flex items-center justify-center gap-2 rounded-xl border-2 py-5 text-lg font-medium transition-colors"
          :class="
            judgeAnswer === 'false'
              ? 'border-danger bg-danger/10 text-danger'
              : 'border-default-200 text-default-500 hover:border-default-300'
          "
          @click="judgeAnswer = 'false'"
        >
          <KunIcon name="lucide:circle-x" />错误
        </button>
      </div>
    </template>

    <!-- fill -->
    <template v-else-if="type === 'fill'">
      <div class="space-y-2">
        <label class="text-sm font-medium">
          每个空的可接受答案 (回车添加, 判分时忽略大小写与空格)
        </label>
        <div v-for="(_, i) in blanks" :key="i" class="flex items-start gap-2">
          <KunTagInput
            v-model="blanks[i]"
            :label="`第 ${i + 1} 空`"
            placeholder="输入一个可接受答案后回车"
            class-name="grow"
          />
          <KunButton
            :is-icon-only="true"
            variant="light"
            color="danger"
            class-name="mt-6"
            :disabled="blanks.length <= 1"
            @click="removeBlank(i)"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </div>
        <KunButton variant="flat" size="sm" @click="addBlank">
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:plus" />添加空
          </span>
        </KunButton>
      </div>
    </template>

    <!-- essay -->
    <template v-else-if="type === 'essay'">
      <KunTextarea
        v-model="essayReference"
        label="参考答案 (问答题不自动判分, 仅在作答后展示此参考答案)"
        :rows="4"
        placeholder="请填写这道题的参考答案"
        :maxlength="2000"
        :show-char-count="true"
        auto-grow
      />
    </template>
  </div>
</template>
