// Wire types for the Galgame 题库 (quiz) feature. Field names are snake_case to
// match the Go DTOs verbatim (see apps/api/internal/galgame/dto/quiz_dto.go).

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

export interface QuizGalgameBrief {
  id: number
  content_limit: string
  name: KunLanguage
}

export interface QuizStats {
  view: number
  answer_count: number
  correct_count: number
  quality_average: number
  quality_count: number
}

export interface GalgameQuizCard extends QuizStats {
  id: number
  user: KunUser
  category: QuizCategory
  type: QuizType
  difficulty: number
  question: string
  created: string | Date
  updated: string | Date
  // null when the quiz is general trivia (not bound to a specific game).
  galgame: QuizGalgameBrief | null
}

export interface QuizListPage {
  quiz_data: GalgameQuizCard[]
  total: number
}

// ── content payloads ──────────────────────────────────────────────
// The PUBLIC (pre-answer) payload keeps only what's needed to answer — the
// correct-answer key is stripped server-side.
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

// The FULL payload (revealed only after answering / to the author).
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

// ── submitted answer payloads ─────────────────────────────────────
export type QuizSubmitted =
  | { value: number } // single
  | { values: number[] } // multiple
  | { value: boolean } // judge
  | { values: string[] } // fill
  | { text: string } // essay

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
  type: QuizType
  difficulty: number
  question: string
  content: QuizPublicContent
  created: string | Date
  updated: string | Date
  galgame: QuizGalgameBrief | null
  is_author: boolean
  // Present once the viewer has answered (or is the author) — reveals the key.
  my_answer: QuizAnswerResult | null
}

export interface QuizQualityResult {
  quality_average: number
  quality_count: number
  quality_rating: number
}

// A galgame candidate returned by the 出题 picker search — enriched with a
// banner thumbnail + 会社 (maker) names.
export interface QuizGalgameOption {
  id: number
  name: KunLanguage
  banner: string
  banner_thumbhash?: string
  officials: string[]
}
