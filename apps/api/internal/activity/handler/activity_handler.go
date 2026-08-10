package handler

import (
	"strings"

	"kun-galgame-api/internal/activity/dto"
	"kun-galgame-api/internal/activity/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type ActivityHandler struct {
	activityService *service.ActivityService
}

func NewActivityHandler(activityService *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activityService: activityService}
}

func (h *ActivityHandler) GetActivity(c fiber.Ctx) error {
	var req dto.ActivityRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	res, appErr := h.activityService.GetActivity(c.Context(), req.Type, req.Cursor, req.Limit, utils.IsSFW(c), req.ShowNoResource)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}

func (h *ActivityHandler) GetTab(c fiber.Ctx) error {
	var req dto.TabRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	isSFW := utils.IsSFW(c) || req.ForceSfw

	var res *service.Result
	var appErr *errors.AppError
	if req.Types != "" {
		res, appErr = h.activityService.GetFeedByTypes(c.Context(), strings.Split(req.Types, ","), req.Cursor, req.Limit, isSFW, req.ShowNoResource)
	} else {
		res, appErr = h.activityService.GetTab(c.Context(), req.Tab, req.Cursor, req.Limit, isSFW, req.ShowNoResource)
	}
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}

func (h *ActivityHandler) GetTimeline(c fiber.Ctx) error {
	var req dto.TimelineRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	res, appErr := h.activityService.GetTimeline(c.Context(), req.Cursor, req.Limit, utils.IsSFW(c), req.ShowNoResource)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}
