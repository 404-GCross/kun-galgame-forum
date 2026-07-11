package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/dto"
	"kun-galgame-api/internal/trust/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type TrustHandler struct {
	trustService *service.TrustService
}

func NewTrustHandler(trustService *service.TrustService) *TrustHandler {
	return &TrustHandler{trustService: trustService}
}

// GetReasons returns the report-reason catalog for the report dropdown.
// GET /api/report/reasons
func (h *TrustHandler) GetReasons(c fiber.Ctx) error {
	return response.OK(c, h.trustService.Reasons())
}

// SubmitReport files a report against a content subject on behalf of the
// session user. Generic — the subject kind/id come straight from the body.
// POST /api/report/submit
func (h *TrustHandler) SubmitReport(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.SubmitReportRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	res, appErr := h.trustService.SubmitReport(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}
