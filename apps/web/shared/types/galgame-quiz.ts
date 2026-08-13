export type QuizType = 'single' | 'multiple' | 'judge' | 'fill' | 'essay'
export type QuizCategory =
  | 'plot'
  | 'character'
  | 'system'
  | 'music'
  | 'voice'
  | 'company'
  | 'trivia'
  | 'other'

export type QuizSpoilerLevel = 'none' | 'portion' | 'serious'

export type QuizMyStatus =
  | 'author'
  | 'correct'
  | 'incorrect'
  | 'answered'
  | 'unanswered'

export interface QuizGalgameBrief {
  id: number
  content_limit: string
  name: KunLanguage
}

export interface QuizGalgameDetail {
  id: number
  name: KunLanguage
  content_limit: string
  age_limit: string
  original_language: string
  banner: string
  banner_thumbhash?: string
  officials: string[]
}

export interface QuizStats {
  view: number
  answer_count: number
  correct_count: number
  favorite_count: number
  quality_average: number
  quality_count: number
  comment_count: number
}

export interface GalgameQuizCard extends QuizStats {
  id: number
  user: KunUser
  category: QuizCategory
  spoiler_level: QuizSpoilerLevel
  type: QuizType
  difficulty: number
  question_html: string
  created: string | Date
  updated: string | Date
  status_update_time: string | Date
  my_status: QuizMyStatus
}

export interface QuizListPage {
  quiz_data: GalgameQuizCard[]
  total: number
}

export interface QuizPublicSingle {
  options: string[]
}
export interface QuizPublicMultiple {
  options: string[]
}
export interface QuizPublicFill {
  blank_count: number
}
export type QuizPublicContent =
  | QuizPublicSingle
  | QuizPublicMultiple
  | QuizPublicFill
  | Record<string, never>

export interface QuizFullSingle {
  options: string[]
  answer: number
}
export interface QuizFullMultiple {
  options: string[]
  answers: number[]
}
export interface QuizFullJudge {
  answer: boolean
}
export interface QuizFullFillBlank {
  accepted: string[]
}
export interface QuizFullFill {
  blanks: QuizFullFillBlank[]
}
export interface QuizFullEssay {
  reference: string
}
export type QuizFullContent =
  | QuizFullSingle
  | QuizFullMultiple
  | QuizFullJudge
  | QuizFullFill
  | QuizFullEssay

export type QuizSubmitted =
  | { value: number }
  | { values: number[] }
  | { value: boolean }
  | { values: string[] }
  | { text: string }

export interface QuizAnswerResult {
  submitted: QuizSubmitted | null
  is_correct: boolean | null
  answer: QuizFullContent
  explanation: string
  quality_rating: number | null
  reward_delta: number
}

export interface GalgameQuizPlay extends QuizStats {
  id: number
  user: KunUser
  category: QuizCategory
  spoiler_level: QuizSpoilerLevel
  type: QuizType
  difficulty: number
  question: string
  question_html: string
  content: QuizPublicContent
  description_html: string
  created: string | Date
  updated: string | Date
  hide_galgame: boolean
  galgames: QuizGalgameDetail[]
  is_author: boolean
  is_favorited: boolean
  my_answer: QuizAnswerResult | null
}

export interface QuizQualityResult {
  quality_average: number
  quality_count: number
  quality_rating: number
}

export interface QuizGalgameOption {
  id: number
  name: KunLanguage
  banner: string
  banner_thumbhash?: string
  officials: string[]
}

export interface QuizEditData {
  id: number
  galgame_ids: number[]
  hide_galgame: boolean
  category: QuizCategory
  type: QuizType
  difficulty: number
  spoiler_level: QuizSpoilerLevel
  question: string
  description: string
  content: QuizFullContent
  explanation: string
  galgames: QuizGalgameBrief[]
}

export interface QuizAnswererRecord {
  user: KunUser
  is_correct: boolean | null
  submitted?: QuizSubmitted
  created: string | Date
}
