import { defineStore } from 'pinia'
import { ref } from 'vue'

export const usePersistEditQuizStore = defineStore(
  'KUNGalgameEditQuiz',
  () => {
    const category = ref<QuizCategory>('trivia')
    const type = ref<QuizType>('single')
    const difficulty = ref(3)
    const spoilerLevel = ref<QuizSpoilerLevel>('none')
    const question = ref('')
    const description = ref('')
    const explanation = ref('')
    const showExplanation = ref(false)
    const hideGalgame = ref(false)
    const content = ref<Record<string, unknown>>({})

    const reset = () => {
      category.value = 'trivia'
      type.value = 'single'
      difficulty.value = 3
      spoilerLevel.value = 'none'
      question.value = ''
      description.value = ''
      explanation.value = ''
      showExplanation.value = false
      hideGalgame.value = false
      content.value = {}
    }

    return {
      category,
      type,
      difficulty,
      spoilerLevel,
      question,
      description,
      explanation,
      showExplanation,
      hideGalgame,
      content,
      reset
    }
  },
  {
    persist: {
      storage: piniaPluginPersistedstate.localStorage()
    }
  }
)
