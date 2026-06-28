package middleware

import (
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"

	"github.com/gofiber/fiber/v2"
)

// RequireModerator gates a route to holders of the management capability
// (moderator ⊂ admin ⊂ ren per docs/oauth/11-roles.md). 403 otherwise.
func RequireModerator() fiber.Handler {
	return func(c *fiber.Ctx) error {
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

// RequireAdmin gates a route to holders of the site-administration capability
// (admin ⊂ ren). 403 otherwise.
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
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
