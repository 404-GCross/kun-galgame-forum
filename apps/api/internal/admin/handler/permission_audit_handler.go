package handler

import (
	"strconv"

	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type PermissionAuditHandler struct {
	svc *service.PermissionAuditService
}

func NewPermissionAuditHandler(svc *service.PermissionAuditService) *PermissionAuditHandler {
	return &PermissionAuditHandler{svc: svc}
}

func (h *PermissionAuditHandler) List(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	res, err := h.svc.List(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, errors.ErrInternal("获取权限审计日志失败"))
	}
	return response.OK(c, res)
}
