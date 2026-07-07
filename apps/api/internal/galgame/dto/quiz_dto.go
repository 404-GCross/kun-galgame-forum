package dto

import "encoding/json"

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

// QuizListRequest is the query for GET /galgame-quiz/all. Difficulty/GalgameID/
// UserID default to 0 = "all".
type QuizListRequest struct {
	Page       int    `query:"page" validate:"min=1"`
	Limit      int    `query:"limit" validate:"min=1,max=50"`
	SortField  string `query:"sort_field"`
	SortOrder  string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	Category   string `query:"category"`
	Type       string `query:"type"`
	Difficulty int    `query:"difficulty" validate:"omitempty,min=1,max=10"`
	GalgameID  int    `query:"galgame_id"`
	UserID     int    `query:"user_id"`
}

// CreateQuizRequest is the body of POST /galgame-quiz. `content` is the
// type-specific payload (options + answer key); it is shape-validated in the
// service against `type` before insert.
type CreateQuizRequest struct {
	GalgameID   *int            `json:"galgame_id" validate:"omitempty,min=1"`
	Category    string          `json:"category" validate:"required,oneof=plot character system music voice company trivia other"`
	Type        string          `json:"type" validate:"required,oneof=single multiple judge fill essay"`
	Difficulty  int             `json:"difficulty" validate:"required,min=1,max=10"`
	Question    string          `json:"question" validate:"required,min=1,max=2000"`
	Content     json.RawMessage `json:"content" validate:"required"`
	Explanation string          `json:"explanation" validate:"max=2000"`
}

// AnswerQuizRequest is the body of POST /galgame-quiz/:id/answer. `submitted`
// is the type-specific answer, graded server-side.
type AnswerQuizRequest struct {
	QuizID    int             `json:"quiz_id" validate:"required,min=1"`
	Submitted json.RawMessage `json:"submitted" validate:"required"`
}

// RateQuizQualityRequest is the body of PUT /galgame-quiz/:id/quality.
type RateQuizQualityRequest struct {
	QuizID        int `json:"quiz_id" validate:"required,min=1"`
	QualityRating int `json:"quality_rating" validate:"required,min=1,max=10"`
}

// DeleteQuizRequest is the query for DELETE /galgame-quiz/:id.
type DeleteQuizRequest struct {
	QuizID int `query:"quiz_id" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses — shared
// ──────────────────────────────────────────

// QuizGalgameBrief is the lightweight linked-game info on a quiz. Nil (a
// literal JSON `null`) when the quiz is general trivia.
type QuizGalgameBrief struct {
	ID           int         `json:"id"`
	ContentLimit string      `json:"content_limit"`
	Name         KunLanguage `json:"name"`
}

// QuizStats is the aggregate answer/quality block embedded in card + play.
type QuizStats struct {
	View           int     `json:"view"`
	AnswerCount    int     `json:"answer_count"`
	CorrectCount   int     `json:"correct_count"`
	QualityAverage float64 `json:"quality_average"`
	QualityCount   int     `json:"quality_count"`
}

// ──────────────────────────────────────────
// Responses — list
// ──────────────────────────────────────────

// QuizCard is a single entry in the quiz list response. It never carries the
// `content` payload — cards only show the prompt + meta + stats.
type QuizCard struct {
	ID         int               `json:"id"`
	User       UserBrief         `json:"user"`
	Category   string            `json:"category"`
	Type       string            `json:"type"`
	Difficulty int               `json:"difficulty"`
	Question   string            `json:"question"`
	QuizStats                    // embedded view/answer/correct/quality
	Created    string            `json:"created"`
	Updated    string            `json:"updated"`
	Galgame    *QuizGalgameBrief `json:"galgame"`
}

type QuizListPage struct {
	QuizData []QuizCard `json:"quiz_data"`
	Total    int64      `json:"total"`
}

// ──────────────────────────────────────────
// Responses — detail / play
// ──────────────────────────────────────────

// QuizPlay is the answer view (GET /galgame-quiz/:id). `Content` is the
// STRIPPED public payload — no correct-answer key. `MyAnswer` is non-nil once
// the viewer has answered (or is the author), and only then reveals the full
// answer key + explanation.
type QuizPlay struct {
	ID         int               `json:"id"`
	User       UserBrief         `json:"user"`
	Category   string            `json:"category"`
	Type       string            `json:"type"`
	Difficulty int               `json:"difficulty"`
	Question   string            `json:"question"`
	Content    json.RawMessage   `json:"content"`
	QuizStats                    // embedded view/answer/correct/quality
	Created    string            `json:"created"`
	Updated    string            `json:"updated"`
	Galgame    *QuizGalgameBrief `json:"galgame"`
	IsAuthor   bool              `json:"is_author"`
	MyAnswer   *QuizAnswerResult `json:"my_answer"`
}

// QuizAnswerResult reveals the graded outcome + the full answer key. It is the
// response of POST /galgame-quiz/:id/answer and the `my_answer` block of
// QuizPlay once the viewer has participated.
type QuizAnswerResult struct {
	Submitted     json.RawMessage `json:"submitted"`      // the viewer's answer (null for the author row)
	IsCorrect     *bool           `json:"is_correct"`     // null for essay / author
	Answer        json.RawMessage `json:"answer"`         // full content payload, key revealed
	Explanation   string          `json:"explanation"`    // author's解析, shown post-answer
	QualityRating *int            `json:"quality_rating"` // the viewer's quality vote, null if unrated
	RewardDelta   int             `json:"reward_delta"`   // moemoepoint just granted (answer POST only; 0 on GET)
}

// CreatedQuiz is the response of POST /galgame-quiz — the freshly authored card.
type CreatedQuiz struct {
	ID         int               `json:"id"`
	User       UserBrief         `json:"user"`
	Category   string            `json:"category"`
	Type       string            `json:"type"`
	Difficulty int               `json:"difficulty"`
	Question   string            `json:"question"`
	QuizStats                    // zeroed stats on a fresh quiz
	Created    string            `json:"created"`
	Updated    string            `json:"updated"`
	Galgame    *QuizGalgameBrief `json:"galgame"`
}

// QuizQualityResult is the response of PUT /galgame-quiz/:id/quality.
type QuizQualityResult struct {
	QualityAverage float64 `json:"quality_average"`
	QualityCount   int     `json:"quality_count"`
	QualityRating  int     `json:"quality_rating"`
}
