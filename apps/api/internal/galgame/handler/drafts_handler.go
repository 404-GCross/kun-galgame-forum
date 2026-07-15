package handler

import (
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

// DraftsHandler serves the unclaimed-VNDB-draft list powering the galgame
// detail page's "未发布的游戏" modal. Read-only and SFW-default like the rest of
// the galgame surface — utils.IsSFW reads the content-rating cookie and the
// service maps it to the wiki content_limit (sfw / all). See DraftsService.
type DraftsHandler struct {
	draftsService *service.DraftsService
}

func NewDraftsHandler(draftsService *service.DraftsService) *DraftsHandler {
	return &DraftsHandler{draftsService: draftsService}
}

// GetDrafts — GET /api/galgame/drafts?page=&limit=
//
// Returns {items, total}: unclaimed VNDB drafts (status=2, newest first) as
// enriched GalgameCard[] so the shared frontend card renders each as a claim
// card. Pagination is mandatory (the SFW draft pool is ~43k rows).
func (h *DraftsHandler) GetDrafts(c fiber.Ctx) error {
	page, limit := parseCollectionPage(c, 24)
	pageData, appErr := h.draftsService.GetDrafts(c.Context(), page, limit, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, pageData)
}
