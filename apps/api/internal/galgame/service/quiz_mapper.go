package service

import (
	"math"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/userclient"
)

// quizQualityAverage returns the mean quality vote rounded to one decimal, or
// 0 when there are no votes yet.
func quizQualityAverage(sum, count int) float64 {
	if count <= 0 {
		return 0
	}
	return math.Round(float64(sum)/float64(count)*10) / 10
}

// quizStats builds the embedded stats block from raw counters.
func quizStats(view, answerCount, correctCount, qualitySum, qualityCount int) dto.QuizStats {
	return dto.QuizStats{
		View:           view,
		AnswerCount:    answerCount,
		CorrectCount:   correctCount,
		QualityAverage: quizQualityAverage(qualitySum, qualityCount),
		QualityCount:   qualityCount,
	}
}

// quizGalgameBrief maps a wiki brief into the linked-game DTO.
func quizGalgameBrief(b client.GalgameBrief) *dto.QuizGalgameBrief {
	return &dto.QuizGalgameBrief{
		ID:           b.ID,
		ContentLimit: b.ContentLimit,
		Name: dto.KunLanguage{
			EnUs: b.NameEnUs, JaJp: b.NameJaJp,
			ZhCn: b.NameZhCn, ZhTw: b.NameZhTw,
		},
	}
}

// quizRowToCard maps a quiz row + author + optional linked-game brief to a card.
func quizRowToCard(
	r model.GalgameQuizRow,
	user userclient.User,
	brief *client.GalgameBrief,
) dto.QuizCard {
	card := dto.QuizCard{
		ID:           r.ID,
		User:         userBriefToDTO(user),
		Category:     r.Category,
		SpoilerLevel: r.SpoilerLevel,
		Type:         r.Type,
		Difficulty:   r.Difficulty,
		Question:     r.Question,
		QuizStats:    quizStats(r.View, r.AnswerCount, r.CorrectCount, r.QualitySum, r.QualityCount),
		Created:      r.Created,
		Updated:      r.Updated,
	}
	if brief != nil {
		card.Galgame = quizGalgameBrief(*brief)
	}
	return card
}
