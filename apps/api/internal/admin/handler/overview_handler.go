package handler

import (
	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/middleware"
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

// GetOverview returns counts for all major models.
// GET /api/admin/overview/all
func (h *OverviewHandler) GetOverview(c fiber.Ctx) error {
	// Forward the admin's OAuth Bearer so the wiki /admin/stats merge is
	// authorized; an empty token just degrades the wiki rows to zero (the
	// forum-local counts still render). See OverviewService.GetOverview.
	token := middleware.GetAccessToken(c)
	items, appErr := h.overviewService.GetOverview(c.Context(), token)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// GetStats returns daily counts for the last N days.
// GET /api/admin/overview/stats
func (h *OverviewHandler) GetStats(c fiber.Ctx) error {
	var req dto.GetStatsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	token := middleware.GetAccessToken(c)
	stats, appErr := h.overviewService.GetStats(c.Context(), req.Days, token)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, stats)
}
