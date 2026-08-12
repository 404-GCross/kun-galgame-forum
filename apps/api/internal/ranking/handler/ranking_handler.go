package handler

import (
	"kun-galgame-api/internal/ranking/dto"
	"kun-galgame-api/internal/ranking/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type RankingHandler struct {
	rankingService *service.RankingService
}

func NewRankingHandler(rankingService *service.RankingService) *RankingHandler {
	return &RankingHandler{rankingService: rankingService}
}

func (h *RankingHandler) GetGalgameRanking(c fiber.Ctx) error {
	var req dto.GalgameRankingRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.rankingService.GetGalgameRanking(c.Context(), &req, utils.IsSFW(c)))
}

func (h *RankingHandler) GetTopicRanking(c fiber.Ctx) error {
	var req dto.TopicRankingRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.rankingService.GetTopicRanking(c.Context(), &req, utils.IsSFW(c)))
}

func (h *RankingHandler) GetUserRanking(c fiber.Ctx) error {
	var req dto.UserRankingRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.rankingService.GetUserRanking(c.Context(), &req))
}
