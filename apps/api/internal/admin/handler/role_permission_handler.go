package handler

import (
	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type RolePermissionHandler struct {
	svc *service.RolePermissionService
}

func NewRolePermissionHandler(svc *service.RolePermissionService) *RolePermissionHandler {
	return &RolePermissionHandler{svc: svc}
}

func (h *RolePermissionHandler) GetMatrix(c fiber.Ctx) error {
	matrix, err := h.svc.Matrix(c.Context())
	if err != nil {
		return response.Error(c, errors.ErrInternal("获取角色权限矩阵失败"))
	}
	return response.OK(c, matrix)
}

func (h *RolePermissionHandler) Replace(c fiber.Ctx) error {
	operator, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ReplaceOverridesRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	matrix, appErr := h.svc.ReplaceOverrides(c.Context(), operator.ID, operator.Roles, c.Params("role"), req.Overrides)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, matrix)
}

func (h *RolePermissionHandler) GetBundles(c fiber.Ctx) error {
	bundles := perm.EffectiveBundles()
	out := make(map[string][]string, len(bundles))
	for role, perms := range bundles {
		keys := make([]string, len(perms))
		for i, p := range perms {
			keys[i] = string(p)
		}
		out[role] = keys
	}
	return response.OK(c, out)
}
