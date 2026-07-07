package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

type QuizRepository struct {
	db *gorm.DB
}

func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

// DB exposes the connection for service-owned transactions.
func (r *QuizRepository) DB() *gorm.DB { return r.db }

// quizCardColumns is the explicit column list for list projections — omits the
// heavy `content` JSONB (cards never expose the payload).
const quizCardColumns = `q.id, q.user_id, q.galgame_id, q.category, q.type,
	q.difficulty, q.question, q.view, q.answer_count, q.correct_count,
	q.quality_sum, q.quality_count, q.created, q.updated`

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// FindByID returns the full quiz model (incl. content) or ok=false.
func (r *QuizRepository) FindByID(id int) (*model.GalgameQuiz, bool) {
	var q model.GalgameQuiz
	if err := r.db.First(&q, id).Error; err != nil {
		return nil, false
	}
	return &q, true
}

// FindAnswer returns the (quiz, user) answer row, or ok=false.
func (r *QuizRepository) FindAnswer(quizID, userID int) (*model.GalgameQuizAnswer, bool) {
	var a model.GalgameQuizAnswer
	err := r.db.Where("quiz_id = ? AND user_id = ?", quizID, userID).First(&a).Error
	if err != nil {
		return nil, false
	}
	return &a, true
}

// ListPaginated applies the filter and returns (rows, total).
func (r *QuizRepository) ListPaginated(f model.QuizFilter) ([]model.GalgameQuizRow, int64) {
	query := r.db.Table("galgame_quiz q")
	if f.Category != "" && f.Category != "all" {
		query = query.Where("q.category = ?", f.Category)
	}
	if f.Type != "" && f.Type != "all" {
		query = query.Where("q.type = ?", f.Type)
	}
	if f.Difficulty > 0 {
		query = query.Where("q.difficulty = ?", f.Difficulty)
	}
	if f.GalgameID > 0 {
		query = query.Where("q.galgame_id = ?", f.GalgameID)
	}
	if f.UserID > 0 {
		query = query.Where("q.user_id = ?", f.UserID)
	}

	var total int64
	query.Count(&total)

	orderCol := "q.created"
	switch f.SortField {
	case "view":
		orderCol = "q.view"
	case "difficulty":
		orderCol = "q.difficulty"
	case "answer_count":
		orderCol = "q.answer_count"
	}

	var rows []model.GalgameQuizRow
	query.Select(quizCardColumns).
		Order(orderCol + " " + f.SortOrder).
		Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
		Scan(&rows)
	return rows, total
}

// ListAnsweredByUser returns quizzes the user answered (role='answerer'),
// newest attempt first, paginated.
func (r *QuizRepository) ListAnsweredByUser(userID, page, limit int) ([]model.GalgameQuizRow, int64) {
	base := r.db.Table("galgame_quiz q").
		Joins("JOIN galgame_quiz_answer a ON a.quiz_id = q.id").
		Where("a.user_id = ? AND a.role = ?", userID, "answerer")

	var total int64
	base.Count(&total)

	var rows []model.GalgameQuizRow
	base.Select(quizCardColumns).
		Order("a.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Scan(&rows)
	return rows, total
}

// IncrementView atomically bumps the view counter (best-effort).
func (r *QuizRepository) IncrementView(id int) {
	go r.db.Table("galgame_quiz").Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// Create inserts a new quiz row.
func (r *QuizRepository) Create(tx *gorm.DB, q *model.GalgameQuiz) error {
	return tx.Create(q).Error
}

// CreateAnswer inserts a new answer/roster row.
func (r *QuizRepository) CreateAnswer(tx *gorm.DB, a *model.GalgameQuizAnswer) error {
	return tx.Create(a).Error
}

// BumpAnswerStats increments answer_count (+1) and, when correct, correct_count.
func (r *QuizRepository) BumpAnswerStats(tx *gorm.DB, quizID int, correct bool) error {
	fields := map[string]any{
		"answer_count": gorm.Expr("answer_count + 1"),
	}
	if correct {
		fields["correct_count"] = gorm.Expr("correct_count + 1")
	}
	return tx.Table("galgame_quiz").Where("id = ?", quizID).Updates(fields).Error
}

// AdjustQuality moves quality_sum by sumDelta and quality_count by countDelta.
func (r *QuizRepository) AdjustQuality(tx *gorm.DB, quizID, sumDelta, countDelta int) error {
	return tx.Table("galgame_quiz").Where("id = ?", quizID).Updates(map[string]any{
		"quality_sum":   gorm.Expr("quality_sum + ?", sumDelta),
		"quality_count": gorm.Expr("quality_count + ?", countDelta),
	}).Error
}

// SetAnswerQuality patches the quality_rating on an answer row.
func (r *QuizRepository) SetAnswerQuality(tx *gorm.DB, answerID, rating int) error {
	return tx.Table("galgame_quiz_answer").Where("id = ?", answerID).
		Update("quality_rating", rating).Error
}

// DeleteByID removes a quiz (cascade clears its answer rows).
func (r *QuizRepository) DeleteByID(tx *gorm.DB, id int) error {
	return tx.Where("id = ?", id).Delete(&model.GalgameQuiz{}).Error
}
