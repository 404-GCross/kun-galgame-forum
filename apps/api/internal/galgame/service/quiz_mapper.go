package service

import (
	"math"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/userclient"
)

func quizQualityAverage(sum, count int) float64 {
	if count <= 0 {
		return 0
	}
	return math.Round(float64(sum)/float64(count)*10) / 10
}

func quizStats(view, answerCount, correctCount, favoriteCount, qualitySum, qualityCount, commentCount int) dto.QuizStats {
	return dto.QuizStats{
		View:           view,
		AnswerCount:    answerCount,
		CorrectCount:   correctCount,
		FavoriteCount:  favoriteCount,
		QualityAverage: quizQualityAverage(qualitySum, qualityCount),
		QualityCount:   qualityCount,
		CommentCount:   commentCount,
	}
}

func quizRowToCard(r model.GalgameQuizRow, user userclient.User) dto.QuizCard {
	return dto.QuizCard{
		ID:               r.ID,
		User:             userBriefToDTO(user),
		Category:         r.Category,
		SpoilerLevel:     r.SpoilerLevel,
		Type:             r.Type,
		Difficulty:       r.Difficulty,
		Question:         r.Question,
		QuizStats:        quizStats(r.View, r.AnswerCount, r.CorrectCount, r.FavoriteCount, r.QualitySum, r.QualityCount, r.CommentCount),
		StatusUpdateTime: r.StatusUpdateTime,
		Created:          r.Created,
		Updated:          r.Updated,
	}
}
