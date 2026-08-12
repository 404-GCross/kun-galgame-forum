package handler

import (
	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type OverviewHandler struct {
	overviewService *service.OverviewService
}

func NewOverviewHandler(overviewService *service.OverviewService) *OverviewHandler {
	return &OverviewHandler{overviewService: overviewService}
}

func (h *OverviewHandler) GetOverview(c fiber.Ctx) error {
	items, appErr := h.overviewService.GetOverview(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

func (h *OverviewHandler) GetStats(c fiber.Ctx) error {
	var req dto.GetStatsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	stats, appErr := h.overviewService.GetStats(c.Context(), req.Days)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, stats)
}
