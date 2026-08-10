package handler

import (
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type DraftsHandler struct {
	draftsService *service.DraftsService
}

func NewDraftsHandler(draftsService *service.DraftsService) *DraftsHandler {
	return &DraftsHandler{draftsService: draftsService}
}

func (h *DraftsHandler) GetDrafts(c fiber.Ctx) error {
	page, limit := parseCollectionPage(c, 24)
	filters := service.DraftFilters{
		LabelID:  fiber.Query(c, "official_id", 0),
		TagID:    fiber.Query(c, "tag_id", 0),
		EngineID: fiber.Query(c, "engine_id", 0),
		SeriesID: fiber.Query(c, "series_id", 0),

		OriginalLanguages: fiber.Query(c, "original_language", ""),
	}
	pageData, appErr := h.draftsService.GetDrafts(c.Context(), page, limit, filters)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, pageData)
}
