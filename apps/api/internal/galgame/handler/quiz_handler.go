package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type QuizHandler struct {
	quizService *service.QuizService
}

func NewQuizHandler(quizService *service.QuizService) *QuizHandler {
	return &QuizHandler{quizService: quizService}
}

// GetAllQuizzes — GET /api/galgame-quiz/all
//
// SFW-default: anonymous + cookie-less requests get only quizzes whose linked
// galgame is content_limit=sfw (general-trivia quizzes are always shown).
func (h *QuizHandler) GetAllQuizzes(c fiber.Ctx) error {
	var req dto.QuizListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.quizService.GetAllQuizzes(c.Context(), &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetMyAnswered — GET /api/galgame-quiz/mine/answered (self only)
func (h *QuizHandler) GetMyAnswered(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.QuizListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.quizService.GetMyAnswered(c.Context(), user.ID, req.Page, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// SearchGalgames — GET /api/galgame-quiz/galgame-search
// Name search for the 出题 picker, enriched with banner + 会社.
func (h *QuizHandler) SearchGalgames(c fiber.Ctx) error {
	var req dto.QuizGalgameSearchRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	options := h.quizService.SearchGalgameOptions(
		c.Context(), req.Keywords, utils.IsSFW(c),
	)
	return response.OK(c, options)
}

// GetQuizPlay — GET /api/galgame-quiz/:id
func (h *QuizHandler) GetQuizPlay(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的题目 ID"))
	}
	detail, appErr := h.quizService.GetQuizPlay(c.Context(), id, optionalUID(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// CreateQuiz — POST /api/galgame-quiz
func (h *QuizHandler) CreateQuiz(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateQuizRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	created, appErr := h.quizService.CreateQuiz(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, created)
}

// AnswerQuiz — POST /api/galgame-quiz/:id/answer
func (h *QuizHandler) AnswerQuiz(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.AnswerQuizRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	result, appErr := h.quizService.AnswerQuiz(user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

// RateQuizQuality — PUT /api/galgame-quiz/:id/quality
func (h *QuizHandler) RateQuizQuality(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.RateQuizQualityRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	result, appErr := h.quizService.RateQuizQuality(user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

// DeleteQuiz — DELETE /api/galgame-quiz/:id
func (h *QuizHandler) DeleteQuiz(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.DeleteQuizRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.quizService.DeleteQuiz(user.ID, role.CanModerate(user.Roles), req.QuizID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "题目已删除")
}
