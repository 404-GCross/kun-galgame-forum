package service

import "context"

func quizGateLocked(hideGalgame bool, spoilerLevel string, isAuthor, hasAnswered bool) bool {
	conceals := hideGalgame || (spoilerLevel != "" && spoilerLevel != "none")
	if !conceals {
		return false
	}
	return !isAuthor && !hasAnswered
}

func (s *ResourceCommentService) commentAreaLocked(_ context.Context, src CommentSource, resourceID, viewerID int) bool {
	if src.key != sourceQuiz.key {
		return false
	}

	var row struct {
		UserID       int    `gorm:"column:user_id"`
		HideGalgame  bool   `gorm:"column:hide_galgame"`
		SpoilerLevel string `gorm:"column:spoiler_level"`
	}
	res := s.db.Table("galgame_quiz").
		Select("user_id, hide_galgame, spoiler_level").
		Where("id = ?", resourceID).Limit(1).Find(&row)
	if res.Error != nil {
		return true
	}
	if res.RowsAffected == 0 {
		return true
	}

	isAuthor := viewerID != 0 && viewerID == row.UserID
	return quizGateLocked(row.HideGalgame, row.SpoilerLevel, isAuthor, s.hasAnsweredQuiz(resourceID, viewerID))
}

func (s *ResourceCommentService) hasAnsweredQuiz(quizID, viewerID int) bool {
	if viewerID == 0 {
		return false
	}
	var count int64
	if err := s.db.Table("galgame_quiz_answer").
		Where("quiz_id = ? AND user_id = ?", quizID, viewerID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}
