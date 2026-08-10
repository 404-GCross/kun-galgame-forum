package handler

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type CreatorHandler struct {
	svc *service.CreatorService
}

func NewCreatorHandler(svc *service.CreatorService) *CreatorHandler {
	return &CreatorHandler{svc: svc}
}

func (h *CreatorHandler) Status(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}
	elig, app, isCreator, appErr := h.svc.Status(c.Context(), user.ID, token)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{
		"eligibility": elig,
		"application": app,
		"is_creator":  isCreator,
	})
}

func (h *CreatorHandler) Apply(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}
	var body struct {
		Message string `json:"message"`
	}
	if len(c.Body()) > 0 {
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return response.Error(c, errors.ErrBadRequest("请求体格式错误"))
		}
	}
	app, appErr := h.svc.Apply(c.Context(), user.ID, token, body.Message)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, app)
}
