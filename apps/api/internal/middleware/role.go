package middleware

import (
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"

	"github.com/gofiber/fiber/v3"
)

func RequireModerator() fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return response.Error(c, errors.ErrAuthExpired())
		}
		if !role.CanModerate(user.Roles) {
			return response.Error(c, errors.ErrForbidden("您没有权限进行此操作"))
		}
		return c.Next()
	}
}

func RequireAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return response.Error(c, errors.ErrAuthExpired())
		}
		if !role.CanAdminister(user.Roles) {
			return response.Error(c, errors.ErrForbidden("您没有权限进行此操作"))
		}
		return c.Next()
	}
}

func RequirePermission(p perm.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return response.Error(c, errors.ErrAuthExpired())
		}
		if !perm.CanUser(user.ID, user.Roles, p) {
			return response.Error(c, errors.ErrForbidden("您没有权限进行此操作"))
		}
		return c.Next()
	}
}
